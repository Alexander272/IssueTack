package services

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/Alexander272/IssueTrack/backend/pkg/logger"
	"github.com/Alexander272/IssueTrack/backend/pkg/mattermost"
	"github.com/google/uuid"
)

func (s *MattermostService) StartWSForRealm(ctx context.Context, realmID uuid.UUID) error {
	s.wsMu.Lock()
	defer s.wsMu.Unlock()

	realmKey := realmID.String()

	if _, ok := s.wsClients[realmKey]; ok {
		return nil
	}

	settings, err := s.repo.GetByRealm(ctx, realmID)
	if err != nil {
		return fmt.Errorf("failed to get mattermost settings: %w", err)
	}
	if !settings.IsActive || settings.BotToken == "" {
		return nil
	}

	wsClient := mattermost.NewWSClient(s.most.Client.URL(), settings.BotUserID, func(ctx context.Context, event mattermost.PostedEvent) {
		s.handleWSEvent(ctx, realmID, event)
	})

	s.wsClients[realmKey] = wsClient

	go wsClient.Run(ctx, settings.BotToken)

	logger.Info("mattermost WS client started", logger.StringAttr("realm_id", realmKey))
	return nil
}

func (s *MattermostService) StopWSForRealm(realmID uuid.UUID) {
	s.wsMu.Lock()
	defer s.wsMu.Unlock()

	realmKey := realmID.String()
	if client, ok := s.wsClients[realmKey]; ok {
		client.Stop()
		delete(s.wsClients, realmKey)
		logger.Info("mattermost WS client stopped", logger.StringAttr("realm_id", realmKey))
	}
}

func (s *MattermostService) StopAllWS() {
	s.wsMu.Lock()
	defer s.wsMu.Unlock()

	for key, client := range s.wsClients {
		client.Stop()
		delete(s.wsClients, key)
	}
	logger.Info("all mattermost WS clients stopped")
}

func (s *MattermostService) StartAllActiveWS(ctx context.Context) {
	if s.baseURL == "" {
		logger.Warn("http.base_url is not configured, Mattermost interactive features will not work")
	}

	settings, err := s.repo.GetActive(ctx)
	if err != nil {
		logger.Error("failed to get active mattermost settings for startup", logger.ErrAttr(err))
		return
	}

	for _, st := range settings {
		if err := s.StartWSForRealm(ctx, st.RealmID); err != nil {
			logger.Error("failed to start WS for realm on startup",
				logger.StringAttr("realm_id", st.RealmID.String()),
				logger.ErrAttr(err),
			)
		}
	}
}

func (s *MattermostService) handleWSEvent(ctx context.Context, realmID uuid.UUID, event mattermost.PostedEvent) {
	userID := event.Post.UserId
	channelID := event.ChannelID
	message := event.Post.Message
	fileIDs := event.Post.FileIds

	if userID == "" {
		return
	}

	hasFiles := len(fileIDs) > 0
	if message == "" && !hasFiles {
		return
	}

	logger.Debug("mattermost WS event received",
		logger.StringAttr("realm_id", realmID.String()),
		logger.StringAttr("user_id", userID),
	)

	settings, err := s.repo.GetByRealm(ctx, realmID)
	if err != nil {
		logger.Error("failed to get realm settings for WS event",
			logger.StringAttr("realm_id", realmID.String()),
			logger.ErrAttr(err),
		)
		return
	}

	msg := strings.TrimSpace(message)

	switch {
	case syncCommands.MatchString(msg):
		if err := s.handleSync(ctx, settings, userID, msg); err != nil {
			logger.Error("failed to handle sync from WS",
				logger.StringAttr("user_id", userID),
				logger.ErrAttr(err),
			)
		}

	case createCommands.MatchString(msg):
		if len(event.Post.FileIds) > 0 {
			s.pendingFiles.Store(channelID+":"+userID, event.Post.FileIds)
		}
		if err := s.sendCreateButton(settings.BotToken, channelID, realmID.String()); err != nil {
			logger.Error("failed to send create button from WS",
				logger.StringAttr("user_id", userID),
				logger.ErrAttr(err),
			)
		}

	case helpCommands.MatchString(msg):
		isAdmin := s.checkIsAdmin(ctx, realmID, userID)
		if err := s.sendHelpMessage(settings.BotToken, channelID, isAdmin); err != nil {
			logger.Error("failed to send help message from WS", logger.ErrAttr(err))
		}

	case statusCommands.MatchString(msg):
		resolvedUserID, _, err := s.resolveOrCreateUser(ctx, realmID, userID)
		if err != nil {
			logger.Warn("failed to resolve user for status command",
				logger.StringAttr("mm_user_id", userID),
				logger.ErrAttr(err),
			)
			return
		}
		if err := s.sendStatusMessage(ctx, settings, resolvedUserID, channelID); err != nil {
			logger.Error("failed to send status message from WS", logger.ErrAttr(err))
		}

	case attachCommands.MatchString(msg) && len(event.Post.FileIds) > 0:
		parts := attachCommands.FindStringSubmatch(msg)
		number, _ := strconv.Atoi(parts[1])
		if err := s.handleAttachFiles(ctx, settings, userID, channelID, number, event.Post.FileIds); err != nil {
			logger.Error("failed to handle attach from WS",
				logger.StringAttr("user_id", userID),
				logger.ErrAttr(err),
			)
		}

	default:
		if len(event.Post.FileIds) > 0 {
			if err := s.handleAttachFiles(ctx, settings, userID, channelID, 0, event.Post.FileIds); err != nil {
				logger.Error("failed to handle auto-attach from WS",
					logger.StringAttr("user_id", userID),
					logger.ErrAttr(err),
				)
			}
			return
		}
		isAdmin := s.checkIsAdmin(ctx, realmID, userID)
		if err := s.sendHelpMessage(settings.BotToken, channelID, isAdmin); err != nil {
			logger.Error("failed to send help message from WS", logger.ErrAttr(err))
		}
	}
}
