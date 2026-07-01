package main

import (
	"context"
	"fmt"
	"log"

	"centralized-data-service/config"
	"centralized-data-service/pkgs/database"
)

type MappingRuleV2 struct {
	ID           int64  `gorm:"column:id"`
	SourceField  string `gorm:"column:source_field"`
	TargetColumn string `gorm:"column:target_column"`
}

func (MappingRuleV2) TableName() string {
	return "cdc_system.mapping_rule_v2"
}

func main() {
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	registry := database.NewRegistry(cfg)
	if err := registry.Init(context.Background()); err != nil {
		log.Fatalf("registry init: %v", err)
	}

	dbControl, err := registry.GetDB(database.RoleControlPlane)
	if err != nil {
		log.Fatalf("control-plane db: %v", err)
	}

	var rules []MappingRuleV2
	if err := dbControl.Where("source_object_id = ?", 76).Find(&rules).Error; err != nil {
		log.Fatalf("error query mapping_rule_v2: %v", err)
	}

	fmt.Printf("Mapping rules for source_object_id 76:\n")
	for _, r := range rules {
		fmt.Printf("  SourceField: %s -> TargetColumn: %s\n", r.SourceField, r.TargetColumn)
	}
}
