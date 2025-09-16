package models

type Stats struct {
	Total        int64        `json:"total"`
	Active       int64        `json:"active"`
	Inactive     int64        `json:"inactive"`
	Specialists  int64        `json:"specialists"`
	Clients      int64        `json:"clients"`
	Suspended    int64        `json:"suspended"`
	NewThisMonth int64        `json:"newThisMonth"`
	Growth       *GrowthStats `json:"growth"`
}

type GrowthStats struct {
	Total       float64 `json:"total"`
	Active      float64 `json:"active"`
	Specialists float64 `json:"specialists"`
	Clients     float64 `json:"clients"`
}

type StatsResult struct {
	Total        int64 `bson:"total"`
	Active       int64 `bson:"active"`
	Inactive     int64 `bson:"inactive"`
	Suspended    int64 `bson:"suspended"`
	Specialists  int64 `bson:"specialists"`
	Clients      int64 `bson:"clients"`
	NewThisMonth int64 `bson:"newThisMonth"`
}
