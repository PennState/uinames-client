package uinames

import "fmt"

func Example() {
	client, _ := NewRequest(
		Amount(5),
		ExtraData(),
	)
	resps, _ := client.Get()
	fmt.Println("Response count:", len(resps))
	// Note that responses can be printed or otherwise processed but
	// not in a Go example as the data is random and will fail to match
	// the output comment below.  For example:
	//   for _, resp := range resps {
	// 	   fmt.Printf("Response: %v\n", resp)
	//   }

	// Output:
	// Response count: 5
}
