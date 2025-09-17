package config_test

import (
	"os"

	"github.com/kod2ulz/gostart/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Env Util", func() {

	var helper config.EnvUtil

	BeforeEach(func() {
		os.Clearenv()
	})

	Context("when using the global helper", func() {
		It("should get an existing environment variable", func() {
			os.Setenv("MY_VAR", "my_value")
			Expect(config.Env.GetOrDefault("MY_VAR", "default").String()).To(Equal("my_value"))
		})

		It("should get the default value when the variable is not set", func() {
			Expect(config.Env.GetOrDefault("MY_VAR", "default").String()).To(Equal("default"))
		})
	})

	Context("when using a prefixed helper", func() {
		BeforeEach(func() {
			helper = config.Env.Helper("MY_APP")
		})

		It("should return the correct prefix", func() {
			Expect(helper.Prefix()).To(Equal("MY_APP"))
		})

		It("should get a prefixed environment variable", func() {
			os.Setenv("MY_APP_SETTING", "app_value")
			Expect(helper.Get("SETTING").String()).To(Equal("app_value"))
		})

		It("should get a default value for a prefixed variable", func() {
			Expect(helper.Get("SETTING", "default_val").String()).To(Equal("default_val"))
		})

		It("should get a string value directly", func() {
			os.Setenv("MY_APP_STRING_SETTING", "string_val")
			Expect(helper.GetString("STRING_SETTING", "default")).To(Equal("string_val"))
		})

		Context("MustGet", func() {
			It("should get a value that exists", func() {
				os.Setenv("MY_APP_REQUIRED", "is_here")
				Expect(helper.MustGet("REQUIRED").String()).To(Equal("is_here"))
			})

			It("should panic if the value does not exist", func() {
				Expect(func() { helper.MustGet("REQUIRED") }).To(Panic())
			})
		})

		Context("with key transformation", func() {
			It("should find a dot.case key", func() {
				os.Setenv("MY_APP_SERVER_PORT", "8080")
				Expect(helper.Get("server.port").Int()).To(Equal(8080))
			})

			It("should find a camelCase key", func() {
				os.Setenv("MY_APP_DB_HOST", "localhost")
				Expect(helper.Get("dbHost").String()).To(Equal("localhost"))
			})

			It("should find a kebab-case key", func() {
				os.Setenv("MY_APP_ENABLE_FEATURE_X", "true")
				Expect(helper.Get("enable-feature-x").Bool()).To(BeTrue())
			})
		})
	})

	Context("Value conversions", func() {
		It("should convert to an integer", func() {
			val := config.Value("123")
			Expect(val.Int()).To(Equal(123))
		})

		It("should convert to a boolean", func() {
			Expect(config.Value("true").Bool()).To(BeTrue())
			Expect(config.Value("false").Bool()).To(BeFalse())
		})

		It("should split into a string list", func() {
			val := config.Value("a,b,c")
			Expect(val.StringList(",")).To(Equal([]string{"a", "b", "c"}))
		})
	})
})
