package config_test

import (
	"context"
	"os"

	"github.com/hashicorp/vault/api"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kod2ulz/gostart/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/modules/vault"
	"github.com/testcontainers/testcontainers-go/wait"
	"time"
)

var _ = Describe("Unified Get()", func() {

	// Reset all sources before each test to ensure isolation
	BeforeEach(func() {
		config.Vault = config.VaultSource{}
		config.DB = config.DBSource{}
		config.Yaml = config.YAMLSource{}
		config.Env = config.EnvSource{}
	})

	It("should return the default value when no sources are configured", func() {
		Expect(config.Get("any_key", "default_value").String()).To(Equal("default_value"))
	})

	Context("with Environment variables configured", func() {
		BeforeEach(func() {
			os.Setenv("TEST_KEY", "env_value")
			os.Setenv("ENV_ONLY", "env_is_set")
		})

		AfterEach(func() {
			os.Unsetenv("TEST_KEY")
			os.Unsetenv("ENV_ONLY")
		})

		It("should return a value from the environment", func() {
			Expect(config.Get("ENV_ONLY").String()).To(Equal("env_is_set"))
		})

		It("should prioritize env over default", func() {
			Expect(config.Get("TEST_KEY", "default").String()).To(Equal("env_value"))
		})
	})

	Context("with YAML configured", func() {
		BeforeEach(func() {
			os.Setenv("TEST_KEY", "env_value") // Keep env for precedence test
			yamlContent := `test_key: "yaml_value"
yaml_only: "yaml_is_set"`
			err := os.WriteFile("test.yaml", []byte(yamlContent), 0644)
			Expect(err).NotTo(HaveOccurred())
			Expect(config.Yaml.Load("test.yaml")).To(Succeed())
		})

		AfterEach(func() {
			os.Unsetenv("TEST_KEY")
			os.Remove("test.yaml")
		})

		It("should return a value from yaml", func() {
			Expect(config.Get("yaml_only").String()).To(Equal("yaml_is_set"))
		})

		It("should prioritize yaml over env and default", func() {
			Expect(config.Get("test_key", "default").String()).To(Equal("yaml_value"))
		})
	})

	Context("with Database configured", func() {
		var pgContainer *postgres.PostgresContainer
		var dbPool *pgxpool.Pool
		var ctx = context.Background()

		BeforeEach(func() {
			os.Setenv("TEST_KEY", "env_value") // For precedence test
			// YAML is not loaded here to isolate DB precedence over Env

			var err error
			pgContainer, err = postgres.RunContainer(ctx,
				testcontainers.WithImage("postgres:15-alpine"),
				postgres.WithDatabase("test-db"),
				testcontainers.WithWaitStrategy(
					wait.ForLog("database system is ready to accept connections").
						WithOccurrence(2).WithStartupTimeout(5*time.Second),
				),
			)
			Expect(err).NotTo(HaveOccurred())

			connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
			Expect(err).NotTo(HaveOccurred())

			dbPool, err = pgxpool.New(ctx, connStr)
			Expect(err).NotTo(HaveOccurred())

			_, err = dbPool.Exec(ctx, `CREATE TABLE app_settings (key TEXT PRIMARY KEY, value TEXT)`)
			Expect(err).NotTo(HaveOccurred())

			_, err = dbPool.Exec(ctx, `INSERT INTO app_settings (key, value) VALUES ('test_key', 'db_value'), ('db_only', 'db_is_set')`)
			Expect(err).NotTo(HaveOccurred())

			config.DB.From(dbPool, "app_settings")
		})

		AfterEach(func() {
			os.Unsetenv("TEST_KEY")
			if dbPool != nil {
				dbPool.Close()
			}
			if pgContainer != nil {
				Expect(pgContainer.Terminate(ctx)).To(Succeed())
			}
		})

		It("should return a value from the database", func() {
			Expect(config.Get("db_only").String()).To(Equal("db_is_set"))
		})

		It("should prioritize db over env and default", func() {
			Expect(config.Get("test_key", "default").String()).To(Equal("db_value"))
		})
	})

	Context("with Vault configured", func() {
		var vaultContainer *vault.VaultContainer
		var ctx = context.Background()
		var vaultRootToken = "my-root-token"

		BeforeEach(func() {
			os.Setenv("TEST_KEY", "env_value") // For precedence test

			var err error
			vaultContainer, err = vault.RunContainer(ctx,
				testcontainers.WithImage("hashicorp/vault:1.15"),
				vault.WithToken(vaultRootToken),
				testcontainers.WithWaitStrategy(
					wait.ForLog("Vault server started!").WithOccurrence(1),
				),
			)
			Expect(err).NotTo(HaveOccurred())

			vaultAddr, err := vaultContainer.HttpHostAddress(ctx)
			Expect(err).NotTo(HaveOccurred())

			_, err = config.Vault.Endpoint(vaultAddr, vaultRootToken)
			Expect(err).NotTo(HaveOccurred())

			// Write a secret
			vaultClient, err := api.NewClient(&api.Config{Address: vaultAddr})
			Expect(err).NotTo(HaveOccurred())
			vaultClient.SetToken(vaultRootToken)

			secretData := map[string]interface{}{"data": map[string]interface{}{"value": "vault_value"}}
			_, err = vaultClient.Logical().Write("secret/data/test", secretData)
			Expect(err).NotTo(HaveOccurred())

			secretData2 := map[string]interface{}{"data": map[string]interface{}{"value": "vault_override"}}
			_, err = vaultClient.Logical().Write("secret/data/test_key", secretData2)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			os.Unsetenv("TEST_KEY")
			if vaultContainer != nil {
				Expect(vaultContainer.Terminate(ctx)).To(Succeed())
			}
		})

		It("should return a value from vault", func() {
			Expect(config.Get("secret/data/test.value").String()).To(Equal("vault_value"))
		})

		It("should prioritize vault over all other sources", func() {
			Expect(config.Get("secret/data/test_key.value").String()).To(Equal("vault_override"))
		})
	})
})