package datastructure

//DataBuffer byte slice and length,从pool里取出来，然后装入不同长度的内容之后放入队列
type DataBuffer struct {
	data   []byte
	length int
}
