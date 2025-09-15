package config_test

import (
	"context"
	"time"

	"github.com/hashicorp/vault/api"
	"github.com/kod2ulz/gostart/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/vault"
	"github.com/testcontainers/testcontainers-go/wait"
)

var _ = Describe("Vault Configuration", func() {
	var vaultContainer *vault.VaultContainer
	var vaultClient *api.Client
	var ctx context.Context
	var vaultRootToken = "my-root-token"

	Context("without cache", func() {
		BeforeEach(func() {
			ctx = context.Background()
			var err error
			vaultContainer, err = vault.RunContainer(ctx,
				testcontainers.WithImage("hashicorp/vault:1.15"),
				vault.WithToken(vaultRootToken),
				testcontainers.WithWaitStrategy(
					wait.ForLog("Vault server started!").WithOccurrence(1).WithStartupTimeout(10*time.Second),
				),
			)
			Expect(err).NotTo(HaveOccurred())

			vaultAddr, err := vaultContainer.HttpHostAddress(ctx)
			Expect(err).NotTo(HaveOccurred())

			conf := api.DefaultConfig()
			conf.Address = vaultAddr
			vaultClient, err = api.NewClient(conf)
			Expect(err).NotTo(HaveOccurred())
			vaultClient.SetToken(vaultRootToken)

			// Write a secret
			secretData := map[string]interface{}{"data": map[string]interface{}{"db_pass": "supersecret"}}
			_, err = vaultClient.Logical().Write("secret/data/app/db", secretData)
			Expect(err).NotTo(HaveOccurred())

			// Configure our package
			_, err = config.Vault.Endpoint(vaultAddr, vaultRootToken)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			Expect(vaultContainer.Terminate(ctx)).To(Succeed())
		})

		It("should retrieve a secret from Vault", func() {
			val := config.Vault.Get("secret/data/app/db.db_pass")
			Expect(val.String()).To(Equal("supersecret"))
		})

		It("should return a default value for a nonexistent secret", func() {
			val := config.Vault.Get("secret/data/app/db.nonexistent", "default_pass")
			Expect(val.String()).To(Equal("default_pass"))
		})
	})

	Context("with cache enabled", func() {
		BeforeEach(func() {
			ctx = context.Background()
			var err error
			vaultContainer, err = vault.RunContainer(ctx,
				testcontainers.WithImage("hashicorp/vault:1.15"),
				vault.WithToken(vaultRootToken),
				testcontainers.WithWaitStrategy(
					wait.ForLog("Vault server started!").WithOccurrence(1).WithStartupTimeout(10*time.Second),
				),
			)
			Expect(err).NotTo(HaveOccurred())

			vaultAddr, err := vaultContainer.HttpHostAddress(ctx)
			Expect(err).NotTo(HaveOccurred())

			conf := api.DefaultConfig()
			conf.Address = vaultAddr
			vaultClient, err = api.NewClient(conf)
			Expect(err).NotTo(HaveOccurred())
			vaultClient.SetToken(vaultRootToken)

			// Write a secret
			secretData := map[string]interface{}{"data": map[string]interface{}{"db_pass": "supersecret"}}
			_, err = vaultClient.Logical().Write("secret/data/app/db", secretData)
			Expect(err).NotTo(HaveOccurred())

			// Configure our package with TTL before calling Endpoint
			config.Vault.WithCacheTTL(100 * time.Millisecond)
			_, err = config.Vault.Endpoint(vaultAddr, vaultRootToken)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			Expect(vaultContainer.Terminate(ctx)).To(Succeed())
		})

		It("should use the cache", func() {
			// First call, should hit Vault
			val1 := config.Vault.Get("secret/data/app/db.db_pass")
			Expect(val1.String()).To(Equal("supersecret"))

			// Update the secret in Vault
			updatedSecret := map[string]interface{}{"data": map[string]interface{}{"db_pass": "new_secret"}}
			_, err := vaultClient.Logical().Write("secret/data/app/db", updatedSecret)
			Expect(err).NotTo(HaveOccurred())

			// Second call, should get the cached value
			val2 := config.Vault.Get("secret/data/app/db.db_pass")
			Expect(val2.String()).To(Equal("supersecret"))

			// Wait for cache to expire
			time.Sleep(150 * time.Millisecond)

			// Third call, should get the new value
			val3 := config.Vault.Get("secret/data/app/db.db_pass")
			Expect(val3.String()).To(Equal("new_secret"))
		})
	})
})