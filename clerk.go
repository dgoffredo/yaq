package yaq

// TODO doc all the things
type clerk struct {
	Enqueue        chan messageSend
	Dequeue        chan messageReceive
	Registry       *Registry
	Queue          string
	RootDirectory  string
	queueDirectory string
}

func (clerk *clerk) Run() {
	clerk.queueDirectory = "TODO"
	// TODO
}
