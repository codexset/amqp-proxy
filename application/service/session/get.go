package session

import (
	"amqp-proxy/application/service/session/utils"
	"github.com/google/uuid"
	"github.com/streadway/amqp"
)

func (c *Session) Get(queue string) (receipt string, body []byte, err error) {
	if queue == "" {
		err = QueueNotExists
		return
	}
	var channel *amqp.Channel
	if channel, err = c.conn.Channel(); err != nil {
		return
	}
	var q amqp.Queue
	if q, err = channel.QueueInspect(queue); err != nil {
		channel.Close()
		return
	}
	if q.Messages == 0 {
		channel.Close()
		err = QueueIsEmpty
		return
	}
	notifyClose := make(chan *amqp.Error)
	channel.NotifyClose(notifyClose)
	var msg amqp.Delivery
	var ok bool
	if msg, ok, err = channel.Get(queue, false); err != nil {
		channel.Close()
		go c.logging(Log{
			Queue:   queue,
			Receipt: nil,
			Payload: nil,
			Action:  "Get",
		}, err)
		return
	}
	if ok == false {
		channel.Close()
		err = QueueNotExists
		go c.logging(Log{
			Queue:   queue,
			Receipt: nil,
			Payload: nil,
			Action:  "Get",
		}, err)
		return
	}
	receipt = uuid.New().String()
	c.receipt.Put(receipt, &utils.Option{
		Queue:    queue,
		Channel:  channel,
		Delivery: &msg,
	})
	body = msg.Body
	go func() {
		select {
		case <-notifyClose:
			c.receipt.Remove(receipt)
			break
		}
	}()
	go c.logging(Log{
		Queue:   queue,
		Receipt: receipt,
		Payload: string(body),
		Action:  "Get",
	})
	return
}
