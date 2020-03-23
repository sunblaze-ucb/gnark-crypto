// Code generated by internal/pairing DO NOT EDIT
package bls381

import (
	"testing"

	"github.com/consensys/gurvy/bls381/fp"
	"github.com/consensys/gurvy/bls381/fr"
)

func TestPairingLineEval(t *testing.T) {

	G := G2Jac{}
	G.X.SetString("2397924191342060428599195401003679400662079238241504450774193330269781506569475895482852526619995975970137974219624",
		"3253138172346762597134335826030291720742705836670663536332107353602145032374655754310365517037438478482781832800086")
	G.Y.SetString("1232241749649797261837110431657148239498502323360832970162662297113649358894158740956353620591860223421095077422231",
		"2699122556714635885354874561737498546399697761274203268065174091190801698152035059896456812958527101011144523472425")
	G.Z.SetString("1",
		"0")

	H := G2Jac{}
	H.X.SetString("626386381354423724675748134644035068069768833090929585240121521648404986563376676132557273114885584225291158274311",
		"339602966943385098532843838069492450190193142238058987746928451075713850258993345145697326485194377986534880517016")
	H.Y.SetString("986571722191489040873388299452948995357368227934165555457446233106695491199835414186175924320043202212317117490883",
		"1407835426138544332721794179872521644798595673903353560732315176812738120660736175621232703037754216510457327346272")
	H.Z.SetString("1",
		"0")

	var a, b, c fp.Element
	a.SetString("2903903751748121992039561169443592957526674295618607189912579965473824470836812596944859552608502931201741951820932")
	b.SetString("1774816561618860752500414710493623341591800940439525170025341193002058157919964682036775870087093646544275868761902")
	c.SetString("1")
	P := G1Jac{}
	P.X = a
	P.Y = b
	P.Z = c

	var Paff G1Affine
	P.ToAffineFromJac(&Paff)

	lRes := &lineEvalRes{}
	lineEvalJac(G, H, &Paff, lRes)

	var expectedA, expectedB, expectedC e2
	expectedA.SetString("2746816319525720818573317398714695290610786823787257703017876624172190671587655383907329814290177195519600649042869",
		"643747286337692864778296894475735847270769031664837266992781597080477672082027893021637700221187514825471134004686")
	expectedB.SetString("1512667481834352587995424361032288920192919642890472182077688617437094168773236832254318567984667090256426031170030",
		"502929900293525728804918427460651455380232209121276695128622351338781664526073250775747424595295099476807695762289")
	expectedC.SetString("2940618921424824052585512724878093813439080759436307077649341963761289470985090932191084231645712795631291954885371",
		"2306463453623604673276907490306214197464628223215034760146285194077185211280172629647779515842943264699205272612043")

	if !lRes.r1.Equal(&expectedA) {
		t.Fatal("Error A coeff")
	}
	if !lRes.r0.Equal(&expectedB) {
		t.Fatal("Error A coeff")
	}
	if !lRes.r2.Equal(&expectedC) {
		t.Fatal("Error A coeff")
	}
}

func TestMagicPairing(t *testing.T) {

	var r1, r2 e12

	r1.SetRandom()
	r2.SetRandom()

	t.Log(r1)
	t.Log(r2)

	curve := BLS381()

	res1 := curve.FinalExponentiation(&r1)
	res2 := curve.FinalExponentiation(&r2)

	if res1.Equal(&res2) {
		t.Fatal("TestMagicPairing failed")
	}
}

func TestComputePairing(t *testing.T) {

	curve := BLS381()

	G := curve.g2Gen.Clone()
	P := curve.g1Gen.Clone()
	sG := G.Clone()
	sP := P.Clone()

	var Gaff, sGaff G2Affine
	var Paff, sPaff G1Affine

	// checking bilinearity

	// check 1
	scalar := fr.Element{123}
	sG.ScalarMul(curve, sG, scalar)
	sP.ScalarMul(curve, sP, scalar)

	var mRes1, mRes2, mRes3 e12

	P.ToAffineFromJac(&Paff)
	sP.ToAffineFromJac(&sPaff)
	G.ToAffineFromJac(&Gaff)
	sG.ToAffineFromJac(&sGaff)

	res1 := curve.FinalExponentiation(curve.MillerLoop(Paff, sGaff, &mRes1))
	res2 := curve.FinalExponentiation(curve.MillerLoop(sPaff, Gaff, &mRes2))

	if !res1.Equal(&res2) {
		t.Fatal("pairing failed")
	}

	// check 2
	s1G := G.Clone()
	s2G := G.Clone()
	s3G := G.Clone()
	s1 := fr.Element{29372983}
	s2 := fr.Element{209302420904}
	var s3 fr.Element
	s3.Add(&s1, &s2)
	s1G.ScalarMul(curve, s1G, s1)
	s2G.ScalarMul(curve, s2G, s2)
	s3G.ScalarMul(curve, s3G, s3)

	var s1Gaff, s2Gaff, s3Gaff G2Affine
	s1G.ToAffineFromJac(&s1Gaff)
	s2G.ToAffineFromJac(&s2Gaff)
	s3G.ToAffineFromJac(&s3Gaff)

	rs1 := curve.FinalExponentiation(curve.MillerLoop(Paff, s1Gaff, &mRes1))
	rs2 := curve.FinalExponentiation(curve.MillerLoop(Paff, s2Gaff, &mRes2))
	rs3 := curve.FinalExponentiation(curve.MillerLoop(Paff, s3Gaff, &mRes3))
	rs1.Mul(&rs2, &rs1)
	if !rs3.Equal(&rs1) {
		t.Fatal("pairing failed2")
	}

}

//--------------------//
//     benches		  //
//--------------------//

func BenchmarkLineEval(b *testing.B) {

	curve := BLS381()

	H := G2Jac{}
	H.ScalarMul(curve, &curve.g2Gen, fr.Element{1213})

	lRes := &lineEvalRes{}
	var g1GenAff G1Affine
	curve.g1Gen.ToAffineFromJac(&g1GenAff)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lineEvalJac(curve.g2Gen, H, &g1GenAff, lRes)
	}

}

func BenchmarkPairing(b *testing.B) {

	curve := BLS381()

	var mRes e12

	var g1GenAff G1Affine
	var g2GenAff G2Affine

	curve.g1Gen.ToAffineFromJac(&g1GenAff)
	curve.g2Gen.ToAffineFromJac(&g2GenAff)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		curve.FinalExponentiation(curve.MillerLoop(g1GenAff, g2GenAff, &mRes))
	}
}

func BenchmarkFinalExponentiation(b *testing.B) {

	var a e12

	curve := BLS381()

	a.SetString(
		"1382424129690940106527336948935335363935127549146605398842626667204683483408227749",
		"0121296909401065273369489353353639351275491466053988426266672046834834082277499690",
		"7336948129690940106527336948935335363935127549146605398842626667204683483408227749",
		"6393512129690940106527336948935335363935127549146605398842626667204683483408227749",
		"2581296909401065273369489353353639351275491466053988426266672046834834082277496644",
		"5331296909401065273369489353353639351275491466053988426266672046834834082277495363",
		"1296909401065273369489353353639351275491466053988426266672046834834082277491382424",
		"0129612969094010652733694893533536393512754914660539884262666720468348340822774990",
		"7336948129690940106527336948935335363935127549146605398842626667204683483408227749",
		"6393129690940106527336948935335363935127549146605398842626667204683483408227749512",
		"2586641296909401065273369489353353639351275491466053988426266672046834834082277494",
		"5312969094010652733694893533536393512754914660539884262666720468348340822774935363")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		curve.FinalExponentiation(&a)
	}

}