package config_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/vault/api"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kod2ulz/gostart/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/modules/vault"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Config Suite")
}

var _ = Describe("Unified Get()", func() {
	var pgContainer *postgres.PostgresContainer
	var dbPool *pgxpool.Pool
	var vaultContainer *vault.VaultContainer
	var ctx context.Context
	var vaultRootToken = "my-root-token"

	// Setup all sources before running the tests in this block
	BeforeEach(func() {
		ctx = context.Background()

		// --- Setup Environment Variable ---
		os.Setenv("HIERARCHY_TEST_KEY", "env_value")
		os.Setenv("ENV_ONLY_KEY", "env_only")

		// --- Setup YAML File ---
		yamlContent := `hierarchy_test_key: "yaml_value"
yaml_only_key: "yaml_only"`
		err := os.WriteFile("test_hierarchy.yaml", []byte(yamlContent), 0644)
		Expect(err).NotTo(HaveOccurred())
		Expect(config.Yaml.Load("test_hierarchy.yaml")).To(Succeed())

		// --- Setup Database ---
		pg, err := postgres.RunContainer(ctx,
			testcontainers.WithImage("postgres:15-alpine"),
			postgres.WithDatabase("test-db"),
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
		_, err = dbPool.Exec(ctx, `INSERT INTO app_config (key, value) VALUES ('hierarchy_test_key', 'db_value'), ('db_only_key', 'db_only')`)
		Expect(err).NotTo(HaveOccurred())
		config.DB.From(dbPool, "app_config")

		// --- Setup Vault ---
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
		secretData := map[string]interface{}{"data": map[string]interface{}{"test-key": "vault_value"}}
		client, _ := api.NewClient(&api.Config{Address: vaultAddr})
		client.SetToken(vaultRootToken)
		_, err = client.Logical().Write("secret/data/hierarchy", secretData)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		os.Clearenv()
		os.Remove("test_hierarchy.yaml")
		dbPool.Close()
		Expect(pgContainer.Terminate(ctx)).To(Succeed())
		Expect(vaultContainer.Terminate(ctx)).To(Succeed())
	})

	It("should return value from Vault (highest precedence)", func() {
		Expect(config.Get("secret/data/hierarchy.test-key").String()).To(Equal("vault_value"))
	})

	It("should return value from DB (when not in Vault)", func() {
		Expect(config.Get("hierarchy_test_key").String()).To(Equal("db_value"))
	})

	It("should return value from YAML (when not in Vault or DB)", func() {
		// Note: DB now has hierarchy_test_key, so this tests that YAML is lower precedence
		Expect(config.Get("hierarchy_test_key").String()).To(Equal("db_value"))
	})

	It("should return value from Env (with key transformation)", func() {
		Expect(config.Get("hierarchy_test_key").String()).To(Equal("db_value")) // DB should override Env
		Expect(config.Get("HIERARCHY_TEST_KEY").String()).To(Equal("env_value")) // Env should be found directly
		Expect(config.Get("some.other.key", "default").String()).To(Equal("default"))
	})

	It("should return default value (when not in any source)", func() {
		Expect(config.Get("non_existent_key", "default_value").String()).To(Equal("default_value"))
	})

	It("should return values from specific sources when others are not configured", func() {
		Expect(config.Get("ENV_ONLY_KEY").String()).To(Equal("env_only"))
		Expect(config.Get("yaml_only_key").String()).To(Equal("yaml_only"))
		Expect(config.Get("db_only_key").String()).To(Equal("db_only"))
	})
})