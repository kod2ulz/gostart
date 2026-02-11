package mq_test

import (
	"time"

	"github.com/kod2ulz/gostart/mq"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MQ Interfaces", func() {
	Describe("Message", func() {
		var message mq.Message

		BeforeEach(func() {
			message = mq.Message{
				Body:        []byte("test message body"),
				Headers:     map[string]interface{}{"header1": "value1"},
				RoutingKey:  "test.routing.key",
				ContentType: "application/json",
				Redelivered: true,
			}
		})

		It("should have correct body", func() {
			Expect(message.Body).To(Equal([]byte("test message body")))
		})

		It("should have headers", func() {
			Expect(message.Headers).To(HaveKey("header1"))
			Expect(message.Headers["header1"]).To(Equal("value1"))
		})

		It("should have routing key", func() {
			Expect(message.RoutingKey).To(Equal("test.routing.key"))
		})

		It("should have content type", func() {
			Expect(message.ContentType).To(Equal("application/json"))
		})

		It("should have redelivered flag", func() {
			Expect(message.Redelivered).To(BeTrue())
		})
	})

	Describe("ExchangeOptions", func() {
		var opts mq.ExchangeOptions

		BeforeEach(func() {
			opts = mq.ExchangeOptions{
				Durable:    true,
				AutoDelete: false,
				Internal:   false,
				NoWait:     false,
				Arguments:  map[string]interface{}{"arg1": "value1"},
			}
		})

		It("should set durable flag", func() {
			Expect(opts.Durable).To(BeTrue())
		})

		It("should set auto-delete flag", func() {
			Expect(opts.AutoDelete).To(BeFalse())
		})

		It("should set internal flag", func() {
			Expect(opts.Internal).To(BeFalse())
		})

		It("should set no-wait flag", func() {
			Expect(opts.NoWait).To(BeFalse())
		})

		It("should have arguments", func() {
			Expect(opts.Arguments).To(HaveKey("arg1"))
			Expect(opts.Arguments["arg1"]).To(Equal("value1"))
		})
	})

	Describe("QueueOptions", func() {
		var opts mq.QueueOptions

		BeforeEach(func() {
			opts = mq.QueueOptions{
				Durable:    true,
				AutoDelete: false,
				Exclusive:  false,
				NoWait:     false,
				Arguments:  map[string]interface{}{"arg1": "value1"},
			}
		})

		It("should set durable flag", func() {
			Expect(opts.Durable).To(BeTrue())
		})

		It("should set auto-delete flag", func() {
			Expect(opts.AutoDelete).To(BeFalse())
		})

		It("should set exclusive flag", func() {
			Expect(opts.Exclusive).To(BeFalse())
		})

		It("should set no-wait flag", func() {
			Expect(opts.NoWait).To(BeFalse())
		})

		It("should have arguments", func() {
			Expect(opts.Arguments).To(HaveKey("arg1"))
			Expect(opts.Arguments["arg1"]).To(Equal("value1"))
		})
	})

	Describe("ConsumeOptions", func() {
		var opts mq.ConsumeOptions

		BeforeEach(func() {
			opts = mq.ConsumeOptions{
				AutoAck:   true,
				Exclusive: false,
				NoLocal:   false,
				NoWait:    false,
				Arguments: map[string]interface{}{"arg1": "value1"},
			}
		})

		It("should set auto-ack flag", func() {
			Expect(opts.AutoAck).To(BeTrue())
		})

		It("should set exclusive flag", func() {
			Expect(opts.Exclusive).To(BeFalse())
		})

		It("should set no-local flag", func() {
			Expect(opts.NoLocal).To(BeFalse())
		})

		It("should set no-wait flag", func() {
			Expect(opts.NoWait).To(BeFalse())
		})

		It("should have arguments", func() {
			Expect(opts.Arguments).To(HaveKey("arg1"))
			Expect(opts.Arguments["arg1"]).To(Equal("value1"))
		})
	})

	Describe("WorkerOptions", func() {
		var opts mq.WorkerOptions

		BeforeEach(func() {
			opts = mq.WorkerOptions{
				QueueOptions: mq.QueueOptions{
					Durable:    true,
					AutoDelete: false,
				},
				BindingKeys:   []string{"key1", "key2"},
				ConsumerTag:   "test-consumer",
				PrefetchCount: 10,
				AutoAck:       false,
				Exclusive:     true,
				ErrorHandler:  mockErrorHandler,
			}
		})

		It("should have queue options", func() {
			Expect(opts.QueueOptions.Durable).To(BeTrue())
		})

		It("should have binding keys", func() {
			Expect(opts.BindingKeys).To(Equal([]string{"key1", "key2"}))
		})

		It("should have consumer tag", func() {
			Expect(opts.ConsumerTag).To(Equal("test-consumer"))
		})

		It("should have prefetch count", func() {
			Expect(opts.PrefetchCount).To(Equal(10))
		})

		It("should have auto-ack flag", func() {
			Expect(opts.AutoAck).To(BeFalse())
		})

		It("should have exclusive flag", func() {
			Expect(opts.Exclusive).To(BeTrue())
		})

		It("should have error handler", func() {
			Expect(opts.ErrorHandler).NotTo(BeNil())
		})
	})

	Describe("RetryAction", func() {
		var action mq.RetryAction

		BeforeEach(func() {
			action = mq.RetryAction{
				ShouldRetry: true,
				Delay:       5 * time.Second,
			}
		})

		It("should have retry flag", func() {
			Expect(action.ShouldRetry).To(BeTrue())
		})

		It("should have delay", func() {
			Expect(action.Delay).To(Equal(5 * time.Second))
		})
	})

	Describe("BrokerConfig", func() {
		var config mq.BrokerConfig

		BeforeEach(func() {
			config = mq.BrokerConfig{
				Host:              "localhost",
				Port:              "5672",
				VHost:             "/test",
				Username:          "user",
				Password:          "pass",
				Protocol:          "amqp",
				Heartbeat:         10 * time.Second,
				Timeout:           30 * time.Second,
				MaxRetries:        5,
				RetryDelay:        1 * time.Second,
				BackoffMultiplier: 2.0,
				AdditionalArgs:    map[string]interface{}{"custom": "value"},
			}
		})

		It("should have host", func() {
			Expect(config.Host).To(Equal("localhost"))
		})

		It("should have port", func() {
			Expect(config.Port).To(Equal("5672"))
		})

		It("should have vhost", func() {
			Expect(config.VHost).To(Equal("/test"))
		})

		It("should have credentials", func() {
			Expect(config.Username).To(Equal("user"))
			Expect(config.Password).To(Equal("pass"))
		})

		It("should have protocol", func() {
			Expect(config.Protocol).To(Equal("amqp"))
		})

		It("should have heartbeat", func() {
			Expect(config.Heartbeat).To(Equal(10 * time.Second))
		})

		It("should have timeout", func() {
			Expect(config.Timeout).To(Equal(30 * time.Second))
		})

		It("should have retry configuration", func() {
			Expect(config.MaxRetries).To(Equal(5))
			Expect(config.RetryDelay).To(Equal(1 * time.Second))
			Expect(config.BackoffMultiplier).To(Equal(2.0))
		})

		It("should have additional arguments", func() {
			Expect(config.AdditionalArgs).To(HaveKey("custom"))
			Expect(config.AdditionalArgs["custom"]).To(Equal("value"))
		})
	})
})

// Mock error handler for testing
func mockErrorHandler(msg interface{}, err error) mq.RetryAction {
	return mq.RetryAction{
		ShouldRetry: false,
		Delay:       0,
	}
}
