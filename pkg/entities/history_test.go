package entities

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gythialy/magnet/pkg/utiles"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestDB(t *testing.T) (*gorm.DB, *HistoryDao, func()) {
	f := "./history.db"
	db, err := gorm.Open(sqlite.Open(f), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		t.Fatal(err)
	}

	_ = db.AutoMigrate(&History{})
	db.Debug()
	dao := NewHistoryDao(db)

	cleanup := func() {
		_ = os.Remove(f)
	}

	return db, dao, cleanup
}

func TestHistoryDao_Cache(t *testing.T) {
	_, dao, cleanup := setupTestDB(t)
	defer cleanup()

	var histories []*History
	userId := int64(0)
	now := time.Now()
	for i := userId; i < 10; i++ {
		histories = append(histories, &History{
			UserId:    userId,
			Url:       fmt.Sprintf("https://test.com/content%d", i),
			UpdatedAt: now,
		})
	}

	if err, i := dao.Insert(histories); err != nil {
		t.Fatal(err)
	} else {
		t.Logf("insert %d rows", i)
	}

	data1 := dao.List(userId)
	t.Log(utiles.ToString(data1))

	date1 := now.AddDate(0, 0, -7)
	if err, i := dao.Insert([]*History{{
		UserId:    userId,
		Url:       fmt.Sprintf("https://test.com/content%d", 2),
		UpdatedAt: date1,
	}, {
		UserId:    userId,
		Url:       fmt.Sprintf("https://test.com/content%d", 4),
		UpdatedAt: date1,
	}}); err != nil {
		t.Fatal(err)
	} else {
		t.Logf("insert %d rows", i)
	}
	data2 := dao.List(userId)
	t.Log(utiles.ToString(data2))

	if err := dao.Clean(); err != nil {
		t.Fatal(err)
	}

	data3 := dao.List(userId)
	t.Log(utiles.ToString(data3))
}

func TestHistoryDao_SearchByTitle(t *testing.T) {
	_, dao, cleanup := setupTestDB(t)
	defer cleanup()

	userId := int64(1)
	now := time.Now()

	// Insert test data
	testData := []*History{
		{UserId: userId, Url: "https://test.com/1", Title: "Test Title 1", UpdatedAt: now},
		{UserId: userId, Url: "https://test.com/2", Title: "Another Test", UpdatedAt: now},
		{UserId: userId, Url: "https://test.com/3", Title: "Something Else", UpdatedAt: now},
		{UserId: userId, Url: "https://test.com/4", Title: "Final Test Title", UpdatedAt: now},
		{UserId: userId, Url: "https://test.com/5", Title: "中文测试标题", UpdatedAt: now},
		{UserId: userId, Url: "https://test.com/6", Title: "Another 中文 Test", UpdatedAt: now},
	}

	if err, count := dao.Insert(testData); err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	} else {
		t.Logf("Inserted %d rows", count)
	}

	// Test cases
	testCases := []struct {
		searchTitle string
		expectedLen int
	}{
		{"Test", 4},
		{"Title", 2},
		{"Something", 1},
		{"Nonexistent", 0},
		{"中文", 2},
		{"测试", 1},
		{"标题", 1},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Search for '%s'", tc.searchTitle), func(t *testing.T) {
			results := dao.SearchByTitle(userId, tc.searchTitle)
			if len(results) != tc.expectedLen {
				t.Errorf("Expected %d results, got %d for search title '%s'", tc.expectedLen, len(results), tc.searchTitle)
			}

			for _, result := range results {
				if !containsInsensitive(result.Title, tc.searchTitle) {
					t.Errorf("Result title '%s' does not contain search term '%s'", result.Title, tc.searchTitle)
				}
			}

			t.Logf("Search results for '%s': %s", tc.searchTitle, utiles.ToString(results))
		})
	}
}

func containsInsensitive(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
