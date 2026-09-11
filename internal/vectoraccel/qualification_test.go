package vectoraccel

import "testing"

func TestCompareCandidatesZeroFalseNegative(t *testing.T){
 c:=map[int]struct{}{1:{},2:{}}
 r:=CompareCandidates(c,map[int]struct{}{1:{}})
 if !r.Passed(){t.Fatal("expected pass")}
}
func TestCompareCandidatesRejectsFalseNegative(t *testing.T){
 r:=CompareCandidates(map[int]struct{}{},map[int]struct{}{1:{}})
 if r.Passed(){t.Fatal("expected false negative")}
}
