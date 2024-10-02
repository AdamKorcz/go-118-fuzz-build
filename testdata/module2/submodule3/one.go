package submodule3

var (
	A = smallFunc("B")
)

func smallFunc(param string) string {
	if param == "A"{
		return "a"
	} else if param == "B" {
		return "b"
	} else {
		return "c"
	}
}