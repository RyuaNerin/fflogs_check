package frontend

import (
	"github.com/gin-gonic/gin"
)

func routeRequest(c *gin.Context) {
	ctx := c.Request.Context()

	ws, err := websocketUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		panic(err)
	}
	defer ws.Close()

	q := &queueData{
		conn:    ws,
		context: ctx,
		done:    make(chan struct{}, 1),
	}

	err = ws.ReadJSON(&q.opt)
	if err != nil {
		panic(err)
	}

	q.opt.Context = ctx
	q.opt.Progress = q.Progress

	queueLock.Lock()
	q.Reorder(len(queue))
	if len(queue) == 0 {
		queueWake <- struct{}{}
	}
	queue = append(queue, q)
	queueLock.Unlock()

	select {
	case <-q.done:
	case <-q.context.Done():
		q.lock.Lock()
		q.skip = true
		q.lock.Unlock()
	}
}
