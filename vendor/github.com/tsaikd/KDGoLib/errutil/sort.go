package errutil

func newSorter(factoryMap map[string]ErrorFactory) *sorter {
	data := []ErrorFactory{}
	for _, factory := range factoryMap {
		data = append(data, factory)
	}
	return &sorter{
		data: data,
	}
}

type sorter struct {
	data []ErrorFactory
}

func (t sorter) Len() int {
	return len(t.data)
}

func (t sorter) Swap(i int, j int) {
	t.data[i], t.data[j] = t.data[j], t.data[i]
}

func (t sorter) Less(i int, j int) bool {
	return t.data[i].Name() < t.data[j].Name()
}
