package publisher

import (
	"fmt"
	"log/slog"
	"search-service/update/core"
	"time"

	"github.com/nats-io/nats.go"
)

type NatsPublisher struct {
	subj string
	conn *nats.Conn
	log  *slog.Logger
}

func NewNatsPublisher(address, subj string, log *slog.Logger) (*NatsPublisher, error) {
	nc, err := nats.Connect(address,
		nats.Name("Publisher"),
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(10),
		nats.ReconnectWait(time.Second),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				log.Warn("disconnected from NATS", "error", err)
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Info("reconnected to NATS", "url", nc.ConnectedUrl())
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			log.Info("connection to NATS closed")
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed connect to broker")
	}
	log.Debug("connected to broker as publisher", "address", address, "subject", subj, "url", nc.ConnectedUrl())
	return &NatsPublisher{
		subj: subj,
		conn: nc,
		log:  log,
	}, nil
}

func (np *NatsPublisher) Close() {
	np.conn.Close()
}

func (np *NatsPublisher) Publish(event core.EventType) error {
	if err := np.conn.Publish(np.subj, []byte(event)); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}
	if err := np.conn.Flush(); err != nil {
		return fmt.Errorf("failed to flush: %w", err)
	}
	np.log.Debug("message published successfully", "subject", np.subj, "event", event)
	return nil
}
