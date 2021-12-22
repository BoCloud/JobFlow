package utils

import (
	"fmt"
	"github.com/eapache/queue"
)

type Vertex struct {
	Key      string
	Parents  []*Vertex
	Children []*Vertex
	Value    interface{}
}

type DAG struct {
	Vertexes []*Vertex
}

func (dag *DAG) AddVertex(v *Vertex) {
	dag.Vertexes = append(dag.Vertexes, v)
}

func (dag *DAG) AddEdge(from, to *Vertex) {
	from.Children = append(from.Children, to)

	to.Parents = append(from.Parents, from)
}

func (dag *DAG) BFS(root *Vertex) error {
	q := queue.New()

	visitMap := make(map[string]bool)
	visitMap[root.Key] = true

	q.Add(root)

	for {
		if q.Length() == 0 {
			break
		}
		current := q.Remove().(*Vertex)

		for _, v := range current.Children {
			if v.Key == root.Key {
				return fmt.Errorf("find bad dependency, please check the dependencies of your templates")
			}
			if _, ok := visitMap[v.Key]; !ok {
				visitMap[v.Key] = true
				q.Add(v)
			}
		}
	}

	return nil
}
