package config_test

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kod2ulz/gostart/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)



var _ = Describe("Database Configuration", func() {
	var pgContainer *postgres.PostgresContainer
	var dbPool *pgxpool.Pool
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
		pg, err := postgres.RunContainer(ctx,
			testcontainers.WithImage("postgres:15-alpine"),
			postgres.WithDatabase("test-db"),
			postgres.WithUsername("postgres"),
			postgres.WithPassword("postgres"),
			testcontainers.WithWaitStrategy(
				wait.ForLog("database system is ready to accept connections").
					WithOccurrence(2).WithStartupTimeout(5*time.Second),
			),
		)
		Expect(err).NotTo(HaveOccurred())
		pgContainer = pg

		connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
		Expect(err).NotTo(HaveOccurred())

		dbPool, err = pgxpool.New(ctx, connStr)
		Expect(err).NotTo(HaveOccurred())

		_, err = dbPool.Exec(ctx, `CREATE TABLE app_config (key TEXT PRIMARY KEY, value TEXT)`);
		Expect(err).NotTo(HaveOccurred())

		_, err = dbPool.Exec(ctx, `INSERT INTO app_config (key, value) VALUES ('db.setting', 'db_value'), ('db.port', '5432')`)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		dbPool.Close()
		Expect(pgContainer.Terminate(ctx)).To(Succeed())
	})

	It("should connect to the database and retrieve a value", func() {
		val := config.DB.From(dbPool, "app_config").Get("db.setting")
		Expect(val.String()).To(Equal("db_value"))
	})

	It("should retrieve an integer value", func() {
		val := config.DB.From(dbPool, "app_config").Get("db.port")
		Expect(val.Int()).To(Equal(5432))
	})

	It("should return a default value if key is not found", func() {
		val := config.DB.From(dbPool, "app_config").Get("non.existent", "default_val")
		Expect(val.String()).To(Equal("default_val"))
	})

	It("should use the cache", func() {
		// Set a 1-second cache
		config.DB.From(dbPool, "app_config").WithCache(1 * time.Second)

		// First call, should hit DB
		val1 := config.DB.Get("db.setting")
		Expect(val1.String()).To(Equal("db_value"))

		// Change value in DB
		_, err := dbPool.Exec(ctx, `UPDATE app_config SET value = 'new_db_value' WHERE key = 'db.setting'`)
		Expect(err).NotTo(HaveOccurred())

		// Second call, should still get cached value
		val2 := config.DB.Get("db.setting")
		Expect(val2.String()).To(Equal("db_value"))

		// Wait for cache to expire
		time.Sleep(1100 * time.Millisecond)

		// Third call, should get the new value from DB
		val3 := config.DB.Get("db.setting")
		Expect(val3.String()).To(Equal("new_db_value"))
	})

	It("should use the custom lookup function when provided", func() {
		// Define a custom lookup function for the test
		lookup := func(ctx context.Context, db config.Dbtx, key string) (string, error) {
			// This custom function appends "-custom" to the key and queries
			customKey := key + "-custom"
			query := fmt.Sprintf("SELECT value FROM app_config WHERE key = $1")
			var result string
			err := db.QueryRow(ctx, query, customKey).Scan(&result)
			return result, err
		}

		// Add a custom value to the DB
		_, err := dbPool.Exec(ctx, `INSERT INTO app_config (key, value) VALUES ('my.setting-custom', 'custom_value')`)
		Expect(err).NotTo(HaveOccurred())

		// Configure the DB source to use the custom lookup
		val := config.DB.From(dbPool, "app_config").WithLookup(lookup).Get("my.setting")

		// Expect the value returned by the custom lookup
		Expect(val.String()).To(Equal("custom_value"))
	})
})
