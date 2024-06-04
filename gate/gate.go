package gate

type Gate struct {
	ch chan struct{}
}

// 限制并行数量
func New(num int) *Gate {
	return &Gate{
		ch: make(chan struct{}, num),
	}
}

func (g *Gate) Enter() {
	g.ch <- struct{}{}
}

func (g *Gate) Leave() {
	<-g.ch
}
