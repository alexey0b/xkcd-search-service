package subscriber

import (
	"context"
	"fmt"
	"log/slog"
	"search-service/search/core"
	"time"

	"github.com/nats-io/nats.go"
)

type NatsSubscriber struct {
	conn *nats.Conn
	sub  *nats.Subscription
	log  *slog.Logger
}

func NewNatsSubscriber(address, subj string, handler core.EventHandler, log *slog.Logger) (*NatsSubscriber, error) {
	nc, err := nats.Connect(address,
		nats.Name("Subscriber"),
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

	sub, err := nc.Subscribe(subj, func(msg *nats.Msg) {
		if err := handler.HandleEvent(context.TODO(), core.EventType(msg.Data)); err != nil {
			log.Error("failed to handle event", "error", err)
		} else {
			log.Debug("received message", "subject", subj)
		}
	})
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe on subject %s: %w", subj, err)
	}
	log.Debug("connected to broker as subscriber", "address", address, "subject", subj, "url", nc.ConnectedUrl())
	return &NatsSubscriber{
		conn: nc,
		sub:  sub,
		log:  log,
	}, nil
}

func (ns *NatsSubscriber) Unsubscribe() {
	if err := ns.sub.Unsubscribe(); err != nil {
		ns.log.Warn("failed to unsubscribe", "subject", ns.sub.Subject)
	}
	ns.conn.Close()
}
