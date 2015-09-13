package dictator

import "testing"

func Test_CommandRouter(t *testing.T) {
	router := CommandRouter{}
	h := func(nCtx NodeContext) error {
		return nil

	}
	router.AddHandler("test", h)

	fun, ok := router.FindHandler("test")

	if !ok {
		t.Fatal("Expect to find a handler")
	}

	result := fun(NodeContext{})

	if result != nil {
		t.Fatal(result.Error())
	}

}
