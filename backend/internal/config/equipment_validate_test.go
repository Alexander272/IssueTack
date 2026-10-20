package config

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

// TestEquipmentConfig_ValidateEquipment: интеграция включается тремя id, и частично
// заполненная настройка должна давать предупреждение при старте — иначе ошибка в
// конфиге не видна до первой выдачи, а заявка в 1C:ERP просто молча не создаётся.
func TestEquipmentConfig_ValidateEquipment(t *testing.T) {
	valid := uuid.New().String()

	cases := []struct {
		name string
		conf EquipmentConfig
		want bool // ожидаем ли предупреждение
	}{
		{name: "всё пусто — выключено", conf: EquipmentConfig{}, want: false},
		{name: "всё заполнено", conf: EquipmentConfig{ERPCategoryID: valid, ERPGroupID: valid, ERPSiteID: valid}, want: false},
		{name: "только категория", conf: EquipmentConfig{ERPCategoryID: valid}, want: true},
		{name: "два из трёх", conf: EquipmentConfig{ERPCategoryID: valid, ERPGroupID: valid}, want: true},
		{name: "не uuid", conf: EquipmentConfig{ERPCategoryID: "not-a-uuid", ERPGroupID: valid, ERPSiteID: valid}, want: true},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			warn := tt.conf.ValidateEquipment()
			if tt.want && warn == "" {
				t.Fatal("ожидалось предупреждение, получена пустая строка")
			}
			if !tt.want && warn != "" {
				t.Fatalf("предупреждения быть не должно, получено: %s", warn)
			}
			if tt.want && !strings.Contains(warn, "1C:ERP") {
				t.Fatalf("предупреждение должно называть интеграцию, получено: %s", warn)
			}
		})
	}
}
