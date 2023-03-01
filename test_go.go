package main

import (
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

type Trie struct {
	next    [26]*Trie
	pattern string
}

func (t *Trie) Insert(word string) {
	cur := t
	for i := 0; i < len(word); i++ {
		index := word[i] - 'a'
		if cur.next[index] == nil {
			cur.next[index] = &Trie{
				next:    [26]*Trie{},
				pattern: "",
			}
		}
		cur = cur.next[index]
	}
	cur.pattern = word
}

func (t *Trie) StartWith(prefix string) string {
	cur := t
	for i := 0; i < len(prefix); i++ {
		index := prefix[i] - 'a'
		if cur.next[index] == nil {
			return prefix
		}
		if cur.next[index].pattern != "" {
			return cur.next[index].pattern
		}
		cur = cur.next[index]
	}
	return prefix
}

func replaceWords(dictionary []string, sentence string) string {
	res := []string{}
	trie := &Trie{}
	for _, dict := range dictionary {
		trie.Insert(dict)
	}
	words := strings.Split(sentence, " ")
	for _, word := range words {
		res = append(res, trie.StartWith(word))
	}
	return strings.Join(res, " ")
}
