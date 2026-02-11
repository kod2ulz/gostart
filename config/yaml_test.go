package config_test

import (
	"os"

	"github.com/kod2ulz/gostart/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("YAML Configuration", func() {

	BeforeEach(func() {
		content := `
server:
  port: 8080
  host: "localhost"
database:
  host: "db.example.com"
  port: 5432
  user: "admin"
features:
  new_dashboard:
    enabled: true
    theme: "dark"
`
		err := os.WriteFile("test_config.yaml", []byte(content), 0644)
		Expect(err).NotTo(HaveOccurred())

		err = config.Yaml.Load("test_config.yaml")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		os.Remove("test_config.yaml")
	})

	It("should get a string value", func() {
		Expect(config.Yaml.Get("database.host").String()).To(Equal("db.example.com"))
	})

	It("should get an integer value", func() {
		Expect(config.Yaml.Get("server.port").Int()).To(Equal(8080))
	})

	It("should get a boolean value", func() {
		Expect(config.Yaml.Get("features.new_dashboard.enabled").Bool()).To(BeTrue())
	})

	It("should return a default value if the key does not exist", func() {
		Expect(config.Yaml.Get("server.nonexistent", "default").String()).To(Equal("default"))
	})

	It("should return a default value for a deeper non-existent key", func() {
		Expect(config.Yaml.Get("features.new_dashboard.nonexistent", "default_theme").String()).To(Equal("default_theme"))
	})

	It("should handle intermediate keys that are not maps gracefully", func() {
		Expect(config.Yaml.Get("server.host.sublevel", "default").String()).To(Equal("default"))
	})
})
