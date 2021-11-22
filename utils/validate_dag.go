package utils

import (
	"fmt"
	"github.com/eapache/queue"
)

// 邻接表
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
			fmt.Println("done")
			break
		}
		current := q.Remove().(*Vertex)

		fmt.Println("bfs key", current.Key)

		for _, v := range current.Children {
			fmt.Printf("from:%v to:%s\n", current.Key, v.Key)
			if v.Key == root.Key {
				return fmt.Errorf("find bad dependency, please check the dependencies of your templates; ")
			}
			if _, ok := visitMap[v.Key]; !ok {
				visitMap[v.Key] = true
				q.Add(v)
			}
		}
	}

	return nil
}
