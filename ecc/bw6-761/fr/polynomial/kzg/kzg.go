// Copyright 2020 ConsenSys Software Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by consensys/gnark-crypto DO NOT EDIT

package kzg

import (
	"errors"
	"io"
	"math/big"

	"github.com/consensys/gnark-crypto/ecc/bw6-761"
	"github.com/consensys/gnark-crypto/ecc/bw6-761/fr"
	"github.com/consensys/gnark-crypto/ecc/bw6-761/fr/fft"
	bw6761_pol "github.com/consensys/gnark-crypto/ecc/bw6-761/fr/polynomial"
	fiatshamir "github.com/consensys/gnark-crypto/fiat-shamir"
	"github.com/consensys/gnark-crypto/internal/parallel"
	"github.com/consensys/gnark-crypto/polynomial"
)

var (
	errNbDigestsNeqNbPolynomials = errors.New("number of digests is not the same as the number of polynomials")
	errUnsupportedSize           = errors.New("the size of the polynomials exceeds the capacity of the SRS")
	errDigestNotInG1             = errors.New("the digest is not in G1")
	errProofNotInG1              = errors.New("the proof is not in G1")
)

// Digest commitment of a polynomial
type Digest = bw6761.G1Affine

// Scheme stores KZG data
type Scheme struct {

	// Domain to perform polynomial division. The size of the domain is the lowest power of 2 greater than Size.
	Domain fft.Domain

	// SRS stores the result of the MPC
	SRS struct {
		G1 []bw6761.G1Affine  // [gen [alpha]gen , [alpha**2]gen, ... ]
		G2 [2]bw6761.G2Affine // [gen, [alpha]gen ]
	}
}

// Proof KZG proof for opening at a single point.
type Proof struct {

	// Point at which the polynomial is evaluated
	Point fr.Element

	// ClaimedValue purported value
	ClaimedValue fr.Element

	// H quotient polynomial (f - f(z))/(x-z)
	H bw6761.G1Affine
}

// NewScheme returns a new KZG scheme.
// This should be used for testing purpose only.
func NewScheme(size int, alpha fr.Element) *Scheme {

	s := &Scheme{}

	d := fft.NewDomain(uint64(size), 0, false)
	s.Domain = *d
	s.SRS.G1 = make([]bw6761.G1Affine, size)

	var bAlpha big.Int
	alpha.ToBigIntRegular(&bAlpha)

	_, _, gen1Aff, gen2Aff := bw6761.Generators()
	s.SRS.G1[0] = gen1Aff
	s.SRS.G2[0] = gen2Aff
	s.SRS.G2[1].ScalarMultiplication(&gen2Aff, &bAlpha)

	alphas := make([]fr.Element, size-1)
	alphas[0] = alpha
	for i := 1; i < len(alphas); i++ {
		alphas[i].Mul(&alphas[i-1], &alpha)
	}
	for i := 0; i < len(alphas); i++ {
		alphas[i].FromMont()
	}
	g1s := bw6761.BatchScalarMultiplicationG1(&gen1Aff, alphas)
	copy(s.SRS.G1[1:], g1s)

	return s
}

// Marshal serializes a proof as H||point||claimed_value.
// The point H is not compressed.
func (p *Proof) Marshal() []byte {

	var res [4 * fr.Bytes]byte

	bH := p.H.RawBytes()
	copy(res[:], bH[:])
	be := p.Point.Bytes()
	copy(res[2*fr.Bytes:], be[:])
	be = p.ClaimedValue.Bytes()
	copy(res[3*fr.Bytes:], be[:])

	return res[:]
}

type BatchProofsSinglePoint struct {
	// Point at which the polynomials are evaluated
	Point fr.Element

	// ClaimedValues purported values
	ClaimedValues []fr.Element

	// H quotient polynomial Sum_i gamma**i*(f - f(z))/(x-z)
	H bw6761.G1Affine
}

// Marshal serializes a proof as H||point||claimed_values.
// The point H is not compressed.
func (p *BatchProofsSinglePoint) Marshal() []byte {
	nbClaimedValues := len(p.ClaimedValues)

	// 2 for H, 1 for point, nbClaimedValues for the claimed values
	res := make([]byte, (3+nbClaimedValues)*fr.Bytes)

	bH := p.H.RawBytes()
	copy(res, bH[:])
	be := p.Point.Bytes()
	copy(res[2*fr.Bytes:], be[:])
	offset := 3 * fr.Bytes
	for i := 0; i < nbClaimedValues; i++ {
		be = p.ClaimedValues[i].Bytes()
		copy(res[offset:], be[:])
		offset += fr.Bytes
	}

	return res

}

// WriteTo writes binary encoding of the scheme data.
// It writes only the SRS, the fft fomain is reconstructed
// from it.
func (s *Scheme) WriteTo(w io.Writer) (int64, error) {

	// encode the fft
	n, err := s.Domain.WriteTo(w)
	if err != nil {
		return n, err
	}

	// encode the SRS
	enc := bw6761.NewEncoder(w)

	toEncode := []interface{}{
		&s.SRS.G2[0],
		&s.SRS.G2[1],
		s.SRS.G1,
	}

	for _, v := range toEncode {
		if err := enc.Encode(v); err != nil {
			return n + enc.BytesWritten(), err
		}
	}

	return n + enc.BytesWritten(), nil
}

// ReadFrom decodes KZG data from reader.
// The kzg data should have been encoded using WriteTo.
// Only the points from the SRS are actually encoded in the
// reader, the fft domain is reconstructed from it.
func (s *Scheme) ReadFrom(r io.Reader) (int64, error) {

	// decode the fft
	n, err := s.Domain.ReadFrom(r)
	if err != nil {
		return n, err
	}

	// decode the SRS
	dec := bw6761.NewDecoder(r)

	toDecode := []interface{}{
		&s.SRS.G2[0],
		&s.SRS.G2[1],
		&s.SRS.G1,
	}

	for _, v := range toDecode {
		if err := dec.Decode(v); err != nil {
			return n + dec.BytesRead(), err
		}
	}

	return n + dec.BytesRead(), nil

}

// Commit commits to a polynomial using a multi exponentiation with the SRS.
// It is assumed that the polynomial is in canonical form, in Montgomery form.
func (s *Scheme) Commit(p polynomial.Polynomial) (polynomial.Digest, error) {

	if p.Degree() >= s.Domain.Cardinality {
		return nil, errUnsupportedSize
	}

	var res Digest
	_p := p.(*bw6761_pol.Polynomial)

	// ensure we don't modify p
	pCopy := make(bw6761_pol.Polynomial, s.Domain.Cardinality)
	copy(pCopy, *_p)

	parallel.Execute(len(*_p), func(start, end int) {
		for i := start; i < end; i++ {
			pCopy[i].FromMont()
		}
	})
	res.MultiExp(s.SRS.G1, pCopy)

	return &res, nil
}

// Open computes an opening proof of _p at _val.
// Returns a MockProof, which is an empty interface.
func (s *Scheme) Open(point interface{}, p polynomial.Polynomial) (polynomial.OpeningProof, error) {

	if p.Degree() >= s.Domain.Cardinality {
		panic("[Open] Size of polynomial exceeds the size supported by the scheme")
	}

	// build the proof
	var res Proof
	claimedValue := p.Eval(point)
	res.Point.SetInterface(point)
	res.ClaimedValue.SetInterface(claimedValue)

	// compute H
	_p := p.(*bw6761_pol.Polynomial)
	h := dividePolyByXminusA(s.Domain, *_p, res.ClaimedValue, res.Point)

	// commit to H
	c, err := s.Commit(&h)
	if err != nil {
		return nil, err
	}
	res.H.Set(c.(*bw6761.G1Affine))

	return &res, nil
}

// Verify verifies a KZG opening proof at a single point
func (s *Scheme) Verify(commitment polynomial.Digest, proof polynomial.OpeningProof) error {

	_commitment := commitment.(*bw6761.G1Affine)
	_proof := proof.(*Proof)

	// verify that the committed quotient and the commitment are in the correct subgroup
	subgroupCheck := _proof.H.IsInSubGroup()
	if !subgroupCheck {
		return errProofNotInG1
	}
	subgroupCheck = _commitment.IsInSubGroup()
	if !subgroupCheck {
		return errDigestNotInG1
	}

	// comm(f(a))
	var claimedValueG1Aff bw6761.G1Affine
	var claimedValueBigInt big.Int
	_proof.ClaimedValue.ToBigIntRegular(&claimedValueBigInt)
	claimedValueG1Aff.ScalarMultiplication(&s.SRS.G1[0], &claimedValueBigInt)

	// [f(alpha) - f(a)]G1Jac
	var fminusfaG1Jac, tmpG1Jac bw6761.G1Jac
	fminusfaG1Jac.FromAffine(_commitment)
	tmpG1Jac.FromAffine(&claimedValueG1Aff)
	fminusfaG1Jac.SubAssign(&tmpG1Jac)

	// [-H(alpha)]G1Aff
	var negH bw6761.G1Affine
	negH.Neg(&_proof.H)

	// [alpha-a]G2Jac
	var alphaMinusaG2Jac, genG2Jac, alphaG2Jac bw6761.G2Jac
	var pointBigInt big.Int
	_proof.Point.ToBigIntRegular(&pointBigInt)
	genG2Jac.FromAffine(&s.SRS.G2[0])
	alphaG2Jac.FromAffine(&s.SRS.G2[1])
	alphaMinusaG2Jac.ScalarMultiplication(&genG2Jac, &pointBigInt).
		Neg(&alphaMinusaG2Jac).
		AddAssign(&alphaG2Jac)

	// [alpha-a]G2Aff
	var xminusaG2Aff bw6761.G2Affine
	xminusaG2Aff.FromJacobian(&alphaMinusaG2Jac)

	// [f(alpha) - f(a)]G1Aff
	var fminusfaG1Aff bw6761.G1Affine
	fminusfaG1Aff.FromJacobian(&fminusfaG1Jac)

	// e([-H(alpha)]G1Aff, G2gen).e([-H(alpha)]G1Aff, [alpha-a]G2Aff) ==? 1
	check, err := bw6761.PairingCheck(
		[]bw6761.G1Affine{fminusfaG1Aff, negH},
		[]bw6761.G2Affine{s.SRS.G2[0], xminusaG2Aff},
	)
	if err != nil {
		return err
	}
	if !check {
		return polynomial.ErrVerifyOpeningProof
	}
	return nil
}

// BatchOpenSinglePoint creates a batch opening proof of several polynomials at a single point
func (s *Scheme) BatchOpenSinglePoint(point interface{}, digests []polynomial.Digest, polynomials []polynomial.Polynomial) (polynomial.BatchOpeningProofSinglePoint, error) {

	nbDigests := len(digests)
	if nbDigests != len(polynomials) {
		return nil, errNbDigestsNeqNbPolynomials
	}

	var res BatchProofsSinglePoint

	// compute the purported values
	res.ClaimedValues = make([]fr.Element, len(polynomials))
	for i := 0; i < len(polynomials); i++ {
		res.ClaimedValues[i] = polynomials[i].Eval(point).(fr.Element)
	}

	// set the point at which the evaluation is done
	res.Point.SetInterface(point)

	// derive the challenge gamma, binded to the point and the commitments
	fs := fiatshamir.NewTranscript(fiatshamir.SHA256, "gamma")
	if err := fs.Bind("gamma", res.Point.Marshal()); err != nil {
		return nil, err
	}
	for i := 0; i < len(digests); i++ {
		if err := fs.Bind("gamma", digests[i].Marshal()); err != nil {
			return nil, err
		}
	}
	gammaByte, err := fs.ComputeChallenge("gamma")
	if err != nil {
		return nil, err
	}
	var gamma fr.Element
	gamma.SetBytes(gammaByte)

	// compute sum_i gamma**i*f and sum_i gamma**i*f(a)
	var sumGammaiTimesEval fr.Element
	sumGammaiTimesEval.Set(&res.ClaimedValues[nbDigests-1])
	sumGammaiTimesPol := polynomials[nbDigests-1].Clone()
	for i := nbDigests - 2; i >= 0; i-- {
		sumGammaiTimesEval.Mul(&sumGammaiTimesEval, &gamma).
			Add(&sumGammaiTimesEval, &res.ClaimedValues[i])
		sumGammaiTimesPol.ScaleInPlace(&gamma)
		sumGammaiTimesPol.Add(polynomials[i], sumGammaiTimesPol)
	}

	// compute H
	_sumGammaiTimesPol := sumGammaiTimesPol.(*bw6761_pol.Polynomial)
	h := dividePolyByXminusA(s.Domain, *_sumGammaiTimesPol, sumGammaiTimesEval, res.Point)
	c, err := s.Commit(&h)
	if err != nil {
		return nil, err
	}
	res.H.Set(c.(*bw6761.G1Affine))

	return &res, nil
}

func (s *Scheme) BatchVerifySinglePoint(digests []polynomial.Digest, batchOpeningProof polynomial.BatchOpeningProofSinglePoint) error {

	nbDigests := len(digests)

	_batchOpeningProof := batchOpeningProof.(*BatchProofsSinglePoint)

	// check consistancy between numbers of claims vs number of digests
	if len(digests) != len(_batchOpeningProof.ClaimedValues) {
		return errNbDigestsNeqNbPolynomials
	}

	// subgroup checks for digests and the proof
	subgroupCheck := true
	for i := 0; i < len(digests); i++ {
		_digest := digests[i].(*bw6761.G1Affine)
		subgroupCheck = subgroupCheck && _digest.IsInSubGroup()
	}
	if !subgroupCheck {
		return errDigestNotInG1
	}
	subgroupCheck = subgroupCheck && _batchOpeningProof.H.IsInSubGroup()
	if !subgroupCheck {
		return errProofNotInG1
	}

	// derive the challenge gamma, binded to the point and the commitments
	fs := fiatshamir.NewTranscript(fiatshamir.SHA256, "gamma")
	err := fs.Bind("gamma", _batchOpeningProof.Point.Marshal())
	if err != nil {
		return err
	}
	for i := 0; i < len(digests); i++ {
		err := fs.Bind("gamma", digests[i].Marshal())
		if err != nil {
			return err
		}
	}
	gammaByte, err := fs.ComputeChallenge("gamma")
	if err != nil {
		return err
	}
	var gamma fr.Element
	gamma.SetBytes(gammaByte)

	var sumGammaiTimesEval fr.Element
	sumGammaiTimesEval.Set(&_batchOpeningProof.ClaimedValues[nbDigests-1])
	for i := nbDigests - 2; i >= 0; i-- {
		sumGammaiTimesEval.Mul(&sumGammaiTimesEval, &gamma).
			Add(&sumGammaiTimesEval, &_batchOpeningProof.ClaimedValues[i])
	}

	var sumGammaiTimesEvalBigInt big.Int
	sumGammaiTimesEval.ToBigIntRegular(&sumGammaiTimesEvalBigInt)
	var sumGammaiTimesEvalG1Aff bw6761.G1Affine
	sumGammaiTimesEvalG1Aff.ScalarMultiplication(&s.SRS.G1[0], &sumGammaiTimesEvalBigInt)

	var acc fr.Element
	acc.SetOne()
	gammai := make([]fr.Element, len(digests))
	gammai[0].SetOne().FromMont()
	for i := 1; i < len(digests); i++ {
		acc.Mul(&acc, &gamma)
		gammai[i].Set(&acc).FromMont()
	}
	var sumGammaiTimesDigestsG1Aff bw6761.G1Affine
	_digests := make([]bw6761.G1Affine, len(digests))
	for i := 0; i < len(digests); i++ {
		_digests[i].Set(digests[i].(*bw6761.G1Affine))
	}

	sumGammaiTimesDigestsG1Aff.MultiExp(_digests, gammai)

	// sum_i [gamma**i * (f-f(a))]G1
	var sumGammiDiffG1Aff bw6761.G1Affine
	var t1, t2 bw6761.G1Jac
	t1.FromAffine(&sumGammaiTimesDigestsG1Aff)
	t2.FromAffine(&sumGammaiTimesEvalG1Aff)
	t1.SubAssign(&t2)
	sumGammiDiffG1Aff.FromJacobian(&t1)

	// [alpha-a]G2Jac
	var alphaMinusaG2Jac, genG2Jac, alphaG2Jac bw6761.G2Jac
	var pointBigInt big.Int
	_batchOpeningProof.Point.ToBigIntRegular(&pointBigInt)
	genG2Jac.FromAffine(&s.SRS.G2[0])
	alphaG2Jac.FromAffine(&s.SRS.G2[1])
	alphaMinusaG2Jac.ScalarMultiplication(&genG2Jac, &pointBigInt).
		Neg(&alphaMinusaG2Jac).
		AddAssign(&alphaG2Jac)

	// [alpha-a]G2Aff
	var xminusaG2Aff bw6761.G2Affine
	xminusaG2Aff.FromJacobian(&alphaMinusaG2Jac)

	// [-H(alpha)]G1Aff
	var negH bw6761.G1Affine
	negH.Neg(&_batchOpeningProof.H)

	// check the pairing equation
	check, err := bw6761.PairingCheck(
		[]bw6761.G1Affine{sumGammiDiffG1Aff, negH},
		[]bw6761.G2Affine{s.SRS.G2[0], xminusaG2Aff},
	)
	if err != nil {
		return err
	}
	if !check {
		return polynomial.ErrVerifyBatchOpeningSinglePoint
	}
	return nil
}
