package mattermost

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Alexander272/IssueTrack/backend/pkg/logger"
	"github.com/goccy/go-json"
	"github.com/mattermost/mattermost/server/public/model"
)

type Client struct {
	serverURL  string
	httpClient *http.Client
}

func NewClient(serverURL string) *Client {
	return &Client{
		serverURL: strings.TrimRight(serverURL, "/"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) URL() string {
	return c.serverURL
}

func (c *Client) newAPI(botToken string) *model.Client4 {
	api := model.NewAPIv4Client(c.serverURL)
	api.SetToken(botToken)
	api.HTTPClient = c.httpClient
	return api
}

func (c *Client) OpenDialog(botToken string, req *model.OpenDialogRequest) error {
	api := c.newAPI(botToken)
	_, err := api.OpenInteractiveDialog(context.Background(), *req)
	if err != nil {
		logger.Error("mattermost open dialog failed",
			logger.StringAttr("callback_id", req.Dialog.CallbackId),
			logger.ErrAttr(err),
		)
		return fmt.Errorf("open interactive dialog: %w", err)
	}
	return nil
}

func (c *Client) CreatePost(botToken string, post *model.Post) (*model.Post, error) {
	api := c.newAPI(botToken)
	created, _, err := api.CreatePost(context.Background(), post)
	if err != nil {
		logger.Error("mattermost create post failed",
			logger.StringAttr("channel_id", post.ChannelId),
			logger.ErrAttr(err),
		)
		return nil, fmt.Errorf("create post: %w", err)
	}
	return created, nil
}

func (c *Client) UpdatePost(botToken, postID string, patch *model.PostPatch) error {
	api := c.newAPI(botToken)
	_, _, err := api.PatchPost(context.Background(), postID, patch)
	if err != nil {
		logger.Error("mattermost update post failed",
			logger.StringAttr("post_id", postID),
			logger.ErrAttr(err),
		)
		return fmt.Errorf("patch post: %w", err)
	}
	return nil
}

func (c *Client) DeletePost(botToken, postID string) error {
	api := c.newAPI(botToken)
	_, err := api.DeletePost(context.Background(), postID)
	if err != nil {
		logger.Error("mattermost delete post failed",
			logger.StringAttr("post_id", postID),
			logger.ErrAttr(err),
		)
		return fmt.Errorf("delete post: %w", err)
	}
	return nil
}

func (c *Client) GetDMChannel(botToken, botUserID, targetUserID string) (string, error) {
	api := c.newAPI(botToken)
	channel, _, err := api.CreateDirectChannel(context.Background(), botUserID, targetUserID)
	if err != nil {
		return "", fmt.Errorf("create direct channel: %w", err)
	}
	return channel.Id, nil
}

func (c *Client) SendDM(botToken, botUserID, targetUserID, message string) (*model.Post, error) {
	channelID, err := c.GetDMChannel(botToken, botUserID, targetUserID)
	if err != nil {
		return nil, fmt.Errorf("get DM channel: %w", err)
	}

	api := c.newAPI(botToken)
	created, _, err := api.CreatePost(context.Background(), &model.Post{
		ChannelId: channelID,
		Message:   message,
	})
	if err != nil {
		return nil, fmt.Errorf("create DM post: %w", err)
	}

	return created, nil
}

func (c *Client) UploadFile(botToken, channelID string, fileData []byte, filename string) (*model.FileInfo, error) {
	api := c.newAPI(botToken)
	resp, _, err := api.UploadFile(context.Background(), fileData, channelID, filename)
	if err != nil {
		return nil, fmt.Errorf("upload file: %w", err)
	}
	if len(resp.FileInfos) == 0 {
		return nil, fmt.Errorf("no file info returned")
	}
	return resp.FileInfos[0], nil
}

func (c *Client) GetFileInfo(botToken, fileID string) (*model.FileInfo, error) {
	api := c.newAPI(botToken)
	info, _, err := api.GetFileInfo(context.Background(), fileID)
	if err != nil {
		return nil, fmt.Errorf("get file info: %w", err)
	}
	return info, nil
}

func (c *Client) GetPostFileIDs(botToken, postID string) ([]string, error) {
	api := c.newAPI(botToken)
	fileInfos, _, err := api.GetFileInfosForPost(context.Background(), postID, "")
	if err != nil {
		return nil, fmt.Errorf("get file infos for post: %w", err)
	}
	ids := make([]string, len(fileInfos))
	for i, fi := range fileInfos {
		ids[i] = fi.Id
	}
	return ids, nil
}

type UserInfo struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

func (c *Client) GetMe(botToken string) (*UserInfo, error) {
	api := c.newAPI(botToken)
	user, _, err := api.GetMe(context.Background(), "")
	if err != nil {
		return nil, fmt.Errorf("get me: %w", err)
	}
	return &UserInfo{
		ID:       user.Id,
		Username: user.Username,
	}, nil
}

func (c *Client) GetUser(botToken, userID string) (*model.User, error) {
	api := c.newAPI(botToken)
	user, _, err := api.GetUser(context.Background(), userID, "")
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return user, nil
}

func (c *Client) GetUsersInTeam(botToken, teamID string) ([]*model.User, error) {
	api := c.newAPI(botToken)
	var allUsers []*model.User
	page := 0
	const perPage = 100
	for {
		users, _, err := api.GetUsersInTeam(context.Background(), teamID, page, perPage, "")
		if err != nil {
			return nil, fmt.Errorf("failed to get users in team page %d: %w", page, err)
		}
		for _, u := range users {
			if !u.IsBot {
				allUsers = append(allUsers, u)
			}
		}
		if len(users) < perPage {
			break
		}
		page++
	}
	return allUsers, nil
}

func (c *Client) GetTeamByName(botToken, teamName string) (*model.Team, error) {
	api := c.newAPI(botToken)
	team, _, err := api.GetTeamByName(context.Background(), teamName, "")
	if err != nil {
		return nil, fmt.Errorf("get team by name: %w", err)
	}
	return team, nil
}

func (c *Client) DownloadFile(botToken, fileID string) ([]byte, error) {
	api := c.newAPI(botToken)
	data, _, err := api.DownloadFile(context.Background(), fileID, true)
	if err != nil {
		return nil, fmt.Errorf("download file: %w", err)
	}
	return data, nil
}

func UnmarshalSubmission(data []byte, s *model.SubmitDialogRequest) error {
	return json.Unmarshal(data, s)
}

func (c *Client) GetAllUsers(botToken string) ([]*model.User, error) {
	api := c.newAPI(botToken)
	var allUsers []*model.User
	page := 0
	const perPage = 100
	for {
		users, _, err := api.GetUsers(context.Background(), page, perPage, "")
		if err != nil {
			return nil, fmt.Errorf("failed to get users page %d: %w", page, err)
		}
		for _, u := range users {
			if !u.IsBot {
				allUsers = append(allUsers, u)
			}
		}
		if len(users) < perPage {
			break
		}
		page++
	}
	return allUsers, nil
}
