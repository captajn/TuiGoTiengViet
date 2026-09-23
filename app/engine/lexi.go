// Package engine — pure Go port of the UniKey ukengine (LGPL, Pham Kim Long).
// Faithful translation of ukengine.cpp + inputproc.cpp + mactab.cpp,
// specialized to Unicode (UTF-16) output only.
package engine

type VnLexiName int

const VnlNonVnChar VnLexiName = -1

const (
	VnlA VnLexiName = iota
	Vnl_a
	VnlA1
	Vnl_a1
	VnlA2
	Vnl_a2
	VnlA3
	Vnl_a3
	VnlA4
	Vnl_a4
	VnlA5
	Vnl_a5
	VnlAr
	Vnl_ar
	VnlAr1
	Vnl_ar1
	VnlAr2
	Vnl_ar2
	VnlAr3
	Vnl_ar3
	VnlAr4
	Vnl_ar4
	VnlAr5
	Vnl_ar5
	VnlAb
	Vnl_ab
	VnlAb1
	Vnl_ab1
	VnlAb2
	Vnl_ab2
	VnlAb3
	Vnl_ab3
	VnlAb4
	Vnl_ab4
	VnlAb5
	Vnl_ab5
	VnlB
	Vnl_b
	VnlC
	Vnl_c
	VnlD
	Vnl_d
	VnlDD
	Vnl_dd
	VnlE
	Vnl_e
	VnlE1
	Vnl_e1
	VnlE2
	Vnl_e2
	VnlE3
	Vnl_e3
	VnlE4
	Vnl_e4
	VnlE5
	Vnl_e5
	VnlEr
	Vnl_er
	VnlEr1
	Vnl_er1
	VnlEr2
	Vnl_er2
	VnlEr3
	Vnl_er3
	VnlEr4
	Vnl_er4
	VnlEr5
	Vnl_er5
	VnlF
	Vnl_f
	VnlG
	Vnl_g
	VnlH
	Vnl_h
	VnlI
	Vnl_i
	VnlI1
	Vnl_i1
	VnlI2
	Vnl_i2
	VnlI3
	Vnl_i3
	VnlI4
	Vnl_i4
	VnlI5
	Vnl_i5
	VnlJ
	Vnl_j
	VnlK
	Vnl_k
	VnlL
	Vnl_l
	VnlM
	Vnl_m
	VnlN
	Vnl_n
	VnlO
	Vnl_o
	VnlO1
	Vnl_o1
	VnlO2
	Vnl_o2
	VnlO3
	Vnl_o3
	VnlO4
	Vnl_o4
	VnlO5
	Vnl_o5
	VnlOr
	Vnl_or
	VnlOr1
	Vnl_or1
	VnlOr2
	Vnl_or2
	VnlOr3
	Vnl_or3
	VnlOr4
	Vnl_or4
	VnlOr5
	Vnl_or5
	VnlOh
	Vnl_oh
	VnlOh1
	Vnl_oh1
	VnlOh2
	Vnl_oh2
	VnlOh3
	Vnl_oh3
	VnlOh4
	Vnl_oh4
	VnlOh5
	Vnl_oh5
	VnlP
	Vnl_p
	VnlQ
	Vnl_q
	VnlR
	Vnl_r
	VnlS
	Vnl_s
	VnlT
	Vnl_t
	VnlU
	Vnl_u
	VnlU1
	Vnl_u1
	VnlU2
	Vnl_u2
	VnlU3
	Vnl_u3
	VnlU4
	Vnl_u4
	VnlU5
	Vnl_u5
	VnlUh
	Vnl_uh
	VnlUh1
	Vnl_uh1
	VnlUh2
	Vnl_uh2
	VnlUh3
	Vnl_uh3
	VnlUh4
	Vnl_uh4
	VnlUh5
	Vnl_uh5
	VnlV
	Vnl_v
	VnlW
	Vnl_w
	VnlX
	Vnl_x
	VnlY
	Vnl_y
	VnlY1
	Vnl_y1
	VnlY2
	Vnl_y2
	VnlY3
	Vnl_y3
	VnlY4
	Vnl_y4
	VnlY5
	Vnl_y5
	VnlZ
	Vnl_z
	VnlLastChar // = 186, number of symbols
)

type VowelSeq int

const VsNil VowelSeq = -1

const (
	Vs_a VowelSeq = iota
	Vs_ar
	Vs_ab
	Vs_e
	Vs_er
	Vs_i
	Vs_o
	Vs_or
	Vs_oh
	Vs_u
	Vs_uh
	Vs_y
	Vs_ai
	Vs_ao
	Vs_au
	Vs_ay
	Vs_aru
	Vs_ary
	Vs_eo
	Vs_eu
	Vs_eru
	Vs_ia
	Vs_ie
	Vs_ier
	Vs_iu
	Vs_oa
	Vs_oab
	Vs_oe
	Vs_oi
	Vs_ori
	Vs_ohi
	Vs_ua
	Vs_uar
	Vs_ue
	Vs_uer
	Vs_ui
	Vs_uo
	Vs_uor
	Vs_uoh
	Vs_uu
	Vs_uy
	Vs_uha
	Vs_uhi
	Vs_uho
	Vs_uhoh
	Vs_uhu
	Vs_ye
	Vs_yer
	Vs_ieu
	Vs_ieru
	Vs_oai
	Vs_oay
	Vs_oeo
	Vs_uay
	Vs_uary
	Vs_uoi
	Vs_uou
	Vs_uori
	Vs_uohi
	Vs_uohu
	Vs_uya
	Vs_uye
	Vs_uyer
	Vs_uyu
	Vs_uhoi
	Vs_uhou
	Vs_uhohi
	Vs_uhohu
	Vs_yeu
	Vs_yeru
)

type ConSeq int

const CsNil ConSeq = -1

const (
	CsB ConSeq = iota
	CsC
	CsCh
	CsD
	CsDd
	CsDz
	CsG
	CsGh
	CsGi
	CsGin
	CsK
	CsKh
	CsL
	CsM
	CsN
	CsNg
	CsNgh
	CsNh
	CsP
	CsPh
	CsQ
	CsQu
	CsR
	CsS
	CsT
	CsTh
	CsTr
	CsV
	CsX
)

// key event types
const (
	VneRoofAll = iota
	VneRoofA
	VneRoofE
	VneRoofO
	VneHookAll
	VneHookUO
	VneHookU
	VneHookO
	VneBowl
	VneDd
	VneTone0
	VneTone1
	VneTone2
	VneTone3
	VneTone4
	VneTone5
	VneTelexW
	VneMapChar
	VneEscChar
	VneNormal
	VneCount // = 20
)

// char classification
const (
	UkcVn = iota
	UkcWordBreak
	UkcNonVn
	UkcReset
)

// word forms
const (
	VnwNonVn = iota
	VnwEmpty
	VnwC
	VnwV
	VnwCV
	VnwVC
	VnwCVC
)

// input methods
const (
	ImTelex = iota
	ImVni
	ImViqr
	ImMsVi
	ImUsrKeymap
	ImTelexSimple
	ImTelexVni // Telex letters + VNI digits combined
)

const (
	VnStdCharOffset = 0x10000
	InvalidStdChar  = 0xFFFFFFFF
	TotalVnChars    = 213
	MaxUkEngine     = 128
	EnterChar       = 13
	MaxMacroKeyLen  = 16
	MaxMacroTextLen = 1024
	MaxMacroItems   = 1024
)

// StdVnChar: rune-like internal code.
//
//	< 0x10000           -> raw byte/ASCII
//	0x10000+vnSym       -> Vietnamese char (tone/caps encoded in vnSym index)
//	0x10000+186+i       -> special western chars
type StdVnChar = uint32
