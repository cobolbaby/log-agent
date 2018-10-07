func (this *Watchdog) Listen(taskQueueChan chan fileChangeEven, channel) error {
	watcher, err := NewRecursiveWatcher()
	if err != nil {
		this.logger.Error("[NewRecursiveWatcher]%s", err)
		return err
	}
	defer watcher.Close()

	go watcher.NotifyFsEvent(func(e fsnotify.Event) {
		// TODO:自定义事件
		taskQueueChan <- e
	})

	for _, rule := range this.rules {
		this.logger.Info("Listen Path: %s", rule)
		err := watcher.RecursiveAdd(rule)
		if err != nil {
			this.logger.Error("[RecursiveAdd]%s", err)
			return err
		}
	}

	done := make(chan bool)
	// 如果done中还没放数据，那main挂起，直到放数据为止
	<-done
	return nil
}