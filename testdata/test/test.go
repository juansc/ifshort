package test

import "fmt"

func thing() string {
	a := "hi"
	if a != "" {
		a = "hey"
		fmt.Println(a)
		fmt.Println(a)
	}

	f := "hi"
	if true {
		fmt.Println(f)
	}

	g := "hi"
	if returnBool() {
		fmt.Println(g)
	}

	e := "hi there"

	if true {
		fmt.Println("haha") // Scope B
		if e == "hi" {      // Scope A
			fmt.Println("haha") // Scope B
		}
		fmt.Println(e) // Scope A
	}

	d := "foo"
	if d == "hey" {
		fmt.Println(d)
	}

	//b := a
	c := "hi"
	// Even though "c" is only referenced before the last if statement,
	// it shouldn't count since it's referenced several times.
	fmt.Println(c)
	if c != "" {
		fmt.Println(c + d)
	}

	return a
}

func returnBool() bool {
	return true
}
