package helpers

type Set[E comparable] map[E]struct{}

func SetFromList[E comparable](ls []E) Set[E] {
	s := Set[E]{}
	s.AddMany(ls)
	return s
}

func (s Set[E]) Add(elem E) {
	s[elem] = struct{}{}
}

func (s Set[E]) AddMany(elems []E) {
	for _, elem := range elems {
		s[elem] = struct{}{}
	}
}

func (s Set[E]) Contains(v E) bool {
	_, ok := s[v]
	return ok
}

func (s Set[E]) ToList() []E {
	result := []E{}
	for v := range s {
		result = append(result, v)
	}
	return result
}

func (s Set[E]) Union(s2 Set[E]) {
	s.AddMany(s2.ToList())
}

func (s Set[E]) Size() int {
	return len(s)
}
