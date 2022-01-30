package test

import "fmt"

func thing() string {
	a := "hi"
	if a != "" {
		a = "hey"
		fmt.Println(a)
	}

	d := "foo"
	if d == "hey" {
		fmt.Println(d)
	}

	e := "hi there"
	if true {
		if e == "hi" {
			fmt.Println("haha")
		}
	}

	b := a
	c := "hi"
	// Even though "c" is only referenced before the last if statement,
	// it shouldn't count since it's referenced several times.
	fmt.Println(c)
	if b != "" && c != "" {
		fmt.Println(b + c + d)
	}

	return a + "hi"
}
