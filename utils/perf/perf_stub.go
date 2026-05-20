package perf

type stubMonitor struct{}

func (m stubMonitor) Stop() Stats {
	return Stats{}
}

func startMonitor() Monitor {
	return stubMonitor{}
}
