package service

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"
	"restaurant-api/internal/model"
)

var ErrManagementInput = errors.New("입력값을 확인해 주세요")
var ErrUsageLimit = errors.New("이벤트 전송이 너무 잦습니다")

type ManagementService struct{ db *gorm.DB }

func NewManagementService(db *gorm.DB) *ManagementService { return &ManagementService{db: db} }

func (s *ManagementService) Moderate(id uint, hidden bool, reason, admin string) error {
	reason = strings.TrimSpace(reason)
	if id == 0 || admin == "" || reason == "" || utf8.RuneCountInString(reason) > 200 {
		return ErrManagementInput
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		var rating model.Rating
		if err := tx.First(&rating, id).Error; err != nil {
			return err
		}
		if rating.Hidden == hidden {
			return nil
		}
		now := time.Now().UTC()
		if err := tx.Model(&rating).Updates(map[string]any{"hidden": hidden, "moderation_reason": reason, "moderated_at": now}).Error; err != nil {
			return err
		}
		var aggregate struct {
			Count   int
			Average float64
		}
		if err := tx.Model(&model.Rating{}).Where("restaurant_id = ? AND hidden = ?", rating.RestaurantID, false).
			Select("COUNT(*) AS count, COALESCE(AVG(score),0) AS average").Scan(&aggregate).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.Restaurant{}).Where("id = ?", rating.RestaurantID).
			Updates(map[string]any{"avg_rating": aggregate.Average, "rating_count": aggregate.Count}).Error; err != nil {
			return err
		}
		return tx.Create(&model.ReviewModeration{ReviewID: id, Admin: admin, Hidden: hidden, Reason: reason}).Error
	})
}

type ManagedReview struct {
	ID               uint      `json:"id"`
	RestaurantID     uint      `json:"restaurant_id"`
	Restaurant       string    `json:"restaurant"`
	AuthorName       string    `json:"author_name"`
	Score            int       `json:"score"`
	Comment          string    `json:"comment"`
	Hidden           bool      `json:"hidden"`
	ModerationReason string    `json:"moderation_reason"`
	CreatedAt        time.Time `json:"created_at"`
}

func (s *ManagementService) Reviews(status, search string, page int) ([]ManagedReview, int64, error) {
	if page < 1 || page > 10000 || utf8.RuneCountInString(search) > 80 {
		return nil, 0, ErrManagementInput
	}
	q := s.db.Table("ratings").Joins("JOIN restaurants ON restaurants.id = ratings.restaurant_id").
		Where("ratings.deleted_at IS NULL AND restaurants.deleted_at IS NULL")
	switch status {
	case "hidden":
		q = q.Where("ratings.hidden = ?", true)
	case "visible":
		q = q.Where("ratings.hidden = ?", false)
	case "all", "":
	default:
		return nil, 0, ErrManagementInput
	}
	if search != "" {
		q = q.Where("restaurants.title LIKE ? OR ratings.comment LIKE ?", "%"+search+"%", "%"+search+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	rows := []ManagedReview{}
	err := q.Select("ratings.id, ratings.restaurant_id, restaurants.title AS restaurant, ratings.author_name, ratings.score, ratings.comment, ratings.hidden, ratings.moderation_reason, ratings.created_at").
		Order("ratings.id DESC").Offset((page - 1) * 50).Limit(50).Scan(&rows).Error
	return rows, total, err
}

func (s *ManagementService) Audit() ([]model.ReviewModeration, error) {
	rows := []model.ReviewModeration{}
	err := s.db.Order("id DESC").Limit(50).Find(&rows).Error
	return rows, err
}

func (s *ManagementService) RecordEvent(e model.UsageEvent, now time.Time) error {
	e.Query = strings.TrimSpace(e.Query)
	e.Category = strings.TrimSpace(e.Category)
	e.Source = strings.TrimSpace(e.Source)
	if e.Visitor == "" || len(e.Visitor) > 64 || utf8.RuneCountInString(e.Query) > 80 || utf8.RuneCountInString(e.Category) > 40 || utf8.RuneCountInString(e.Source) > 60 {
		return ErrManagementInput
	}
	switch e.Type {
	case "page_view":
		e.RestaurantID = 0
		e.Query = ""
		e.Category = ""
	case "search":
		if e.Query == "" {
			return ErrManagementInput
		}
		e.RestaurantID = 0
		e.Source = ""
	case "category":
		if e.Category == "" {
			return ErrManagementInput
		}
		e.RestaurantID = 0
		e.Query = ""
		e.Source = ""
	case "detail", "directions", "roulette":
		if e.RestaurantID == 0 {
			return ErrManagementInput
		}
		var count int64
		if err := s.db.Model(&model.Restaurant{}).Where("id = ?", e.RestaurantID).Count(&count).Error; err != nil {
			return err
		}
		if count != 1 {
			return ErrManagementInput
		}
		e.Query = ""
		e.Category = ""
		e.Source = ""
	default:
		return ErrManagementInput
	}
	e.ID = 0
	e.CreatedAt = now.UTC()
	return s.db.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&model.UsageEvent{}).Where("visitor = ? AND created_at >= ?", e.Visitor, now.Add(-time.Minute).UTC()).Count(&count).Error; err != nil {
			return err
		}
		if count >= 120 {
			return ErrUsageLimit
		}
		if err := tx.Where("created_at < ?", now.AddDate(0, 0, -90).UTC()).Delete(&model.UsageEvent{}).Error; err != nil {
			return err
		}
		return tx.Create(&e).Error
	})
}

type MetricRow struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}
type DailyUsage struct {
	Day       string `json:"day"`
	PageViews int64  `json:"page_views"`
	Visitors  int64  `json:"visitors"`
}
type RestaurantUsage struct {
	ID         uint   `json:"id"`
	Title      string `json:"title"`
	Details    int64  `json:"details"`
	Directions int64  `json:"directions"`
	Roulette   int64  `json:"roulette"`
}
type UsageStatistics struct {
	Days          int               `json:"days"`
	Start         string            `json:"start"`
	TrackingSince *time.Time        `json:"tracking_since"`
	PageViews     int64             `json:"page_views"`
	Visitors      int64             `json:"visitors"`
	Searches      int64             `json:"searches"`
	Details       int64             `json:"details"`
	Directions    int64             `json:"directions"`
	Roulette      int64             `json:"roulette"`
	Reviews       int64             `json:"reviews"`
	HiddenReviews int64             `json:"hidden_reviews"`
	Daily         []DailyUsage      `json:"daily"`
	Queries       []MetricRow       `json:"queries"`
	Categories    []MetricRow       `json:"categories"`
	Sources       []MetricRow       `json:"sources"`
	Restaurants   []RestaurantUsage `json:"restaurants"`
}

func (s *ManagementService) Statistics(days int, now time.Time) (UsageStatistics, error) {
	stats := UsageStatistics{Days: days, Daily: []DailyUsage{}, Queries: []MetricRow{}, Categories: []MetricRow{}, Sources: []MetricRow{}, Restaurants: []RestaurantUsage{}}
	if days < 1 || days > 90 {
		return stats, ErrManagementInput
	}
	kst := time.FixedZone("KST", 9*3600)
	local := now.In(kst)
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, kst).AddDate(0, 0, 1-days)
	end := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, kst).AddDate(0, 0, 1)
	stats.Start = start.Format("2006-01-02")
	if err := s.db.Where("created_at < ?", now.AddDate(0, 0, -90).UTC()).Delete(&model.UsageEvent{}).Error; err != nil {
		return stats, err
	}
	var first model.UsageEvent
	err := s.db.Order("created_at").First(&first).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return stats, err
	}
	if err == nil {
		stats.TrackingSince = &first.CreatedAt
	}
	q := func() *gorm.DB {
		return s.db.Model(&model.UsageEvent{}).Where("usage_events.created_at >= ? AND usage_events.created_at < ?", start.UTC(), end.UTC())
	}
	for kind, target := range map[string]*int64{"page_view": &stats.PageViews, "search": &stats.Searches, "detail": &stats.Details, "directions": &stats.Directions, "roulette": &stats.Roulette} {
		if err := q().Where("type = ?", kind).Count(target).Error; err != nil {
			return stats, err
		}
	}
	if err := q().Where("type = ?", "page_view").Distinct("visitor").Count(&stats.Visitors).Error; err != nil {
		return stats, err
	}
	if err := s.db.Model(&model.Rating{}).Where("created_at >= ? AND created_at < ?", start.UTC(), end.UTC()).Count(&stats.Reviews).Error; err != nil {
		return stats, err
	}
	if err := s.db.Model(&model.Rating{}).Where("hidden = ?", true).Count(&stats.HiddenReviews).Error; err != nil {
		return stats, err
	}
	var daily []DailyUsage
	if err := q().Where("type = ?", "page_view").Select("date(created_at,'+9 hours') AS day, COUNT(*) AS page_views, COUNT(DISTINCT visitor) AS visitors").Group("day").Order("day").Scan(&daily).Error; err != nil {
		return stats, err
	}
	byDay := map[string]DailyUsage{}
	for _, d := range daily {
		byDay[d.Day] = d
	}
	for i := 0; i < days; i++ {
		day := start.AddDate(0, 0, i).Format("2006-01-02")
		row := byDay[day]
		row.Day = day
		stats.Daily = append(stats.Daily, row)
	}
	for _, group := range []struct {
		kind, field string
		dest        *[]MetricRow
	}{{"search", "query", &stats.Queries}, {"category", "category", &stats.Categories}, {"page_view", "source", &stats.Sources}} {
		// Field names are constants, never client-controlled SQL.
		if err := q().Where("type = ?", group.kind).Select("\"" + group.field + "\" AS name, COUNT(*) AS count").Group(group.field).Order("count DESC, name").Limit(20).Scan(group.dest).Error; err != nil {
			return stats, err
		}
	}
	err = q().Joins("JOIN restaurants ON restaurants.id = usage_events.restaurant_id AND restaurants.deleted_at IS NULL").
		Select("restaurants.id, restaurants.title, SUM(CASE WHEN type='detail' THEN 1 ELSE 0 END) AS details, SUM(CASE WHEN type='directions' THEN 1 ELSE 0 END) AS directions, SUM(CASE WHEN type='roulette' THEN 1 ELSE 0 END) AS roulette").
		Group("restaurants.id, restaurants.title").Order("details DESC, directions DESC, restaurants.id").Limit(20).Scan(&stats.Restaurants).Error
	return stats, err
}
