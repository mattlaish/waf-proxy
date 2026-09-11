package qualification

type Result struct {
	Sample string
	Coraza []int
	VectorScan []int
	FalseNegative []int
	Passed bool
}

func Compare(sample string, coraza, vectorscan []int) Result {
	have:=map[int]bool{}
	for _,v:=range vectorscan { have[v]=true }
	miss:=[]int{}
	for _,v:=range coraza { if !have[v] { miss=append(miss,v) } }
	return Result{Sample:sample,Coraza:coraza,VectorScan:vectorscan,FalseNegative:miss,Passed:len(miss)==0}
}
