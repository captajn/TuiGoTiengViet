package engine

// ---------- static tables ported verbatim from ukengine.cpp / inputproc.cpp / vnconv/data.cpp ----------

type vowelSeqInfo struct {
	length    int
	complete  int
	conSuffix int // allow consonant suffix
	v         [3]VnLexiName
	sub       [3]VowelSeq
	roofPos   int
	withRoof  VowelSeq
	hookPos   int
	withHook  VowelSeq // hook & bowl
}

var vSeqList = []vowelSeqInfo{
	{1, 1, 1, [3]VnLexiName{Vnl_a, VnlNonVnChar, VnlNonVnChar}, [3]VowelSeq{Vs_a, VsNil, VsNil}, -1, Vs_ar, -1, Vs_ab},
	{1, 1, 1, [3]VnLexiName{Vnl_ar, VnlNonVnChar, VnlNonVnChar}, [3]VowelSeq{Vs_ar, VsNil, VsNil}, 0, VsNil, -1, Vs_ab},
	{1, 1, 1, [3]VnLexiName{Vnl_ab, VnlNonVnChar, VnlNonVnChar}, [3]VowelSeq{Vs_ab, VsNil, VsNil}, -1, Vs_ar, 0, VsNil},
	{1, 1, 1, [3]VnLexiName{Vnl_e, VnlNonVnChar, VnlNonVnChar}, [3]VowelSeq{Vs_e, VsNil, VsNil}, -1, Vs_er, -1, VsNil},
	{1, 1, 1, [3]VnLexiName{Vnl_er, VnlNonVnChar, VnlNonVnChar}, [3]VowelSeq{Vs_er, VsNil, VsNil}, 0, VsNil, -1, VsNil},
	{1, 1, 1, [3]VnLexiName{Vnl_i, VnlNonVnChar, VnlNonVnChar}, [3]VowelSeq{Vs_i, VsNil, VsNil}, -1, VsNil, -1, VsNil},
	{1, 1, 1, [3]VnLexiName{Vnl_o, VnlNonVnChar, VnlNonVnChar}, [3]VowelSeq{Vs_o, VsNil, VsNil}, -1, Vs_or, -1, Vs_oh},
	{1, 1, 1, [3]VnLexiName{Vnl_or, VnlNonVnChar, VnlNonVnChar}, [3]VowelSeq{Vs_or, VsNil, VsNil}, 0, VsNil, -1, Vs_oh},
	{1, 1, 1, [3]VnLexiName{Vnl_oh, VnlNonVnChar, VnlNonVnChar}, [3]VowelSeq{Vs_oh, VsNil, VsNil}, -1, Vs_or, 0, VsNil},
	{1, 1, 1, [3]VnLexiName{Vnl_u, VnlNonVnChar, VnlNonVnChar}, [3]VowelSeq{Vs_u, VsNil, VsNil}, -1, VsNil, -1, Vs_uh},
	{1, 1, 1, [3]VnLexiName{Vnl_uh, VnlNonVnChar, VnlNonVnChar}, [3]VowelSeq{Vs_uh, VsNil, VsNil}, -1, VsNil, 0, VsNil},
	{1, 1, 1, [3]VnLexiName{Vnl_y, VnlNonVnChar, VnlNonVnChar}, [3]VowelSeq{Vs_y, VsNil, VsNil}, -1, VsNil, -1, VsNil},
	{2, 1, 0, [3]VnLexiName{Vnl_a, Vnl_i, VnlNonVnChar}, [3]VowelSeq{Vs_a, Vs_ai, VsNil}, -1, VsNil, -1, VsNil},
	{2, 1, 0, [3]VnLexiName{Vnl_a, Vnl_o, VnlNonVnChar}, [3]VowelSeq{Vs_a, Vs_ao, VsNil}, -1, VsNil, -1, VsNil},
	{2, 1, 0, [3]VnLexiName{Vnl_a, Vnl_u, VnlNonVnChar}, [3]VowelSeq{Vs_a, Vs_au, VsNil}, -1, Vs_aru, -1, VsNil},
	{2, 1, 0, [3]VnLexiName{Vnl_a, Vnl_y, VnlNonVnChar}, [3]VowelSeq{Vs_a, Vs_ay, VsNil}, -1, Vs_ary, -1, VsNil},
	{2, 1, 0, [3]VnLexiName{Vnl_ar, Vnl_u, VnlNonVnChar}, [3]VowelSeq{Vs_ar, Vs_aru, VsNil}, 0, VsNil, -1, VsNil},
	{2, 1, 0, [3]VnLexiName{Vnl_ar, Vnl_y, VnlNonVnChar}, [3]VowelSeq{Vs_ar, Vs_ary, VsNil}, 0, VsNil, -1, VsNil},
	{2, 1, 0, [3]VnLexiName{Vnl_e, Vnl_o, VnlNonVnChar}, [3]VowelSeq{Vs_e, Vs_eo, VsNil}, -1, VsNil, -1, VsNil},
	{2, 0, 0, [3]VnLexiName{Vnl_e, Vnl_u, VnlNonVnChar}, [3]VowelSeq{Vs_e, Vs_eu, VsNil}, -1, Vs_eru, -1, VsNil},
	{2, 1, 0, [3]VnLexiName{Vnl_er, Vnl_u, VnlNonVnChar}, [3]VowelSeq{Vs_er, Vs_eru, VsNil}, 0, VsNil, -1, VsNil},
	{2, 1, 1, [3]VnLexiName{Vnl_i, Vnl_a, VnlNonVnChar}, [3]VowelSeq{Vs_i, Vs_ia, VsNil}, -1, VsNil, -1, VsNil},
	{2, 0, 1, [3]VnLexiName{Vnl_i, Vnl_e, VnlNonVnChar}, [3]VowelSeq{Vs_i, Vs_ie, VsNil}, -1, Vs_ier, -1, VsNil},
	{2, 1, 1, [3]VnLexiName{Vnl_i, Vnl_er, VnlNonVnChar}, [3]VowelSeq{Vs_i, Vs_ier, VsNil}, 1, VsNil, -1, VsNil},
	{2, 1, 0, [3]VnLexiName{Vnl_i, Vnl_u, VnlNonVnChar}, [3]VowelSeq{Vs_i, Vs_iu, VsNil}, -1, VsNil, -1, VsNil},
	{2, 1, 1, [3]VnLexiName{Vnl_o, Vnl_a, VnlNonVnChar}, [3]VowelSeq{Vs_o, Vs_oa, VsNil}, -1, VsNil, -1, Vs_oab},
	{2, 1, 1, [3]VnLexiName{Vnl_o, Vnl_ab, VnlNonVnChar}, [3]VowelSeq{Vs_o, Vs_oab, VsNil}, -1, VsNil, 1, VsNil},
	{2, 1, 1, [3]VnLexiName{Vnl_o, Vnl_e, VnlNonVnChar}, [3]VowelSeq{Vs_o, Vs_oe, VsNil}, -1, VsNil, -1, VsNil},
	{2, 1, 0, [3]VnLexiName{Vnl_o, Vnl_i, VnlNonVnChar}, [3]VowelSeq{Vs_o, Vs_oi, VsNil}, -1, Vs_ori, -1, Vs_ohi},
	{2, 1, 0, [3]VnLexiName{Vnl_or, Vnl_i, VnlNonVnChar}, [3]VowelSeq{Vs_or, Vs_ori, VsNil}, 0, VsNil, -1, Vs_ohi},
	{2, 1, 0, [3]VnLexiName{Vnl_oh, Vnl_i, VnlNonVnChar}, [3]VowelSeq{Vs_oh, Vs_ohi, VsNil}, -1, Vs_ori, 0, VsNil},
	{2, 1, 1, [3]VnLexiName{Vnl_u, Vnl_a, VnlNonVnChar}, [3]VowelSeq{Vs_u, Vs_ua, VsNil}, -1, Vs_uar, -1, Vs_uha},
	{2, 1, 1, [3]VnLexiName{Vnl_u, Vnl_ar, VnlNonVnChar}, [3]VowelSeq{Vs_u, Vs_uar, VsNil}, 1, VsNil, -1, VsNil},
	{2, 0, 1, [3]VnLexiName{Vnl_u, Vnl_e, VnlNonVnChar}, [3]VowelSeq{Vs_u, Vs_ue, VsNil}, -1, Vs_uer, -1, VsNil},
	{2, 1, 1, [3]VnLexiName{Vnl_u, Vnl_er, VnlNonVnChar}, [3]VowelSeq{Vs_u, Vs_uer, VsNil}, 1, VsNil, -1, VsNil},
	{2, 1, 0, [3]VnLexiName{Vnl_u, Vnl_i, VnlNonVnChar}, [3]VowelSeq{Vs_u, Vs_ui, VsNil}, -1, VsNil, -1, Vs_uhi},
	{2, 0, 1, [3]VnLexiName{Vnl_u, Vnl_o, VnlNonVnChar}, [3]VowelSeq{Vs_u, Vs_uo, VsNil}, -1, Vs_uor, -1, Vs_uho},
	{2, 1, 1, [3]VnLexiName{Vnl_u, Vnl_or, VnlNonVnChar}, [3]VowelSeq{Vs_u, Vs_uor, VsNil}, 1, VsNil, -1, Vs_uoh},
	{2, 1, 1, [3]VnLexiName{Vnl_u, Vnl_oh, VnlNonVnChar}, [3]VowelSeq{Vs_u, Vs_uoh, VsNil}, -1, Vs_uor, 1, Vs_uhoh},
	{2, 0, 0, [3]VnLexiName{Vnl_u, Vnl_u, VnlNonVnChar}, [3]VowelSeq{Vs_u, Vs_uu, VsNil}, -1, VsNil, -1, Vs_uhu},
	{2, 1, 1, [3]VnLexiName{Vnl_u, Vnl_y, VnlNonVnChar}, [3]VowelSeq{Vs_u, Vs_uy, VsNil}, -1, VsNil, -1, VsNil},
	{2, 1, 0, [3]VnLexiName{Vnl_uh, Vnl_a, VnlNonVnChar}, [3]VowelSeq{Vs_uh, Vs_uha, VsNil}, -1, VsNil, 0, VsNil},
	{2, 1, 0, [3]VnLexiName{Vnl_uh, Vnl_i, VnlNonVnChar}, [3]VowelSeq{Vs_uh, Vs_uhi, VsNil}, -1, VsNil, 0, VsNil},
	{2, 0, 1, [3]VnLexiName{Vnl_uh, Vnl_o, VnlNonVnChar}, [3]VowelSeq{Vs_uh, Vs_uho, VsNil}, -1, VsNil, 0, Vs_uhoh},
	{2, 1, 1, [3]VnLexiName{Vnl_uh, Vnl_oh, VnlNonVnChar}, [3]VowelSeq{Vs_uh, Vs_uhoh, VsNil}, -1, VsNil, 0, VsNil},
	{2, 1, 0, [3]VnLexiName{Vnl_uh, Vnl_u, VnlNonVnChar}, [3]VowelSeq{Vs_uh, Vs_uhu, VsNil}, -1, VsNil, 0, VsNil},
	{2, 0, 1, [3]VnLexiName{Vnl_y, Vnl_e, VnlNonVnChar}, [3]VowelSeq{Vs_y, Vs_ye, VsNil}, -1, Vs_yer, -1, VsNil},
	{2, 1, 1, [3]VnLexiName{Vnl_y, Vnl_er, VnlNonVnChar}, [3]VowelSeq{Vs_y, Vs_yer, VsNil}, 1, VsNil, -1, VsNil},
	{3, 0, 0, [3]VnLexiName{Vnl_i, Vnl_e, Vnl_u}, [3]VowelSeq{Vs_i, Vs_ie, Vs_ieu}, -1, Vs_ieru, -1, VsNil},
	{3, 1, 0, [3]VnLexiName{Vnl_i, Vnl_er, Vnl_u}, [3]VowelSeq{Vs_i, Vs_ier, Vs_ieru}, 1, VsNil, -1, VsNil},
	{3, 1, 0, [3]VnLexiName{Vnl_o, Vnl_a, Vnl_i}, [3]VowelSeq{Vs_o, Vs_oa, Vs_oai}, -1, VsNil, -1, VsNil},
	{3, 1, 0, [3]VnLexiName{Vnl_o, Vnl_a, Vnl_y}, [3]VowelSeq{Vs_o, Vs_oa, Vs_oay}, -1, VsNil, -1, VsNil},
	{3, 1, 0, [3]VnLexiName{Vnl_o, Vnl_e, Vnl_o}, [3]VowelSeq{Vs_o, Vs_oe, Vs_oeo}, -1, VsNil, -1, VsNil},
	{3, 0, 0, [3]VnLexiName{Vnl_u, Vnl_a, Vnl_y}, [3]VowelSeq{Vs_u, Vs_ua, Vs_uay}, -1, Vs_uary, -1, VsNil},
	{3, 1, 0, [3]VnLexiName{Vnl_u, Vnl_ar, Vnl_y}, [3]VowelSeq{Vs_u, Vs_uar, Vs_uary}, 1, VsNil, -1, VsNil},
	{3, 0, 0, [3]VnLexiName{Vnl_u, Vnl_o, Vnl_i}, [3]VowelSeq{Vs_u, Vs_uo, Vs_uoi}, -1, Vs_uori, -1, Vs_uhoi},
	{3, 0, 0, [3]VnLexiName{Vnl_u, Vnl_o, Vnl_u}, [3]VowelSeq{Vs_u, Vs_uo, Vs_uou}, -1, VsNil, -1, Vs_uhou},
	{3, 1, 0, [3]VnLexiName{Vnl_u, Vnl_or, Vnl_i}, [3]VowelSeq{Vs_u, Vs_uor, Vs_uori}, 1, VsNil, -1, Vs_uohi},
	{3, 0, 0, [3]VnLexiName{Vnl_u, Vnl_oh, Vnl_i}, [3]VowelSeq{Vs_u, Vs_uoh, Vs_uohi}, -1, Vs_uori, 1, Vs_uhohi},
	{3, 0, 0, [3]VnLexiName{Vnl_u, Vnl_oh, Vnl_u}, [3]VowelSeq{Vs_u, Vs_uoh, Vs_uohu}, -1, VsNil, 1, Vs_uhohu},
	{3, 1, 0, [3]VnLexiName{Vnl_u, Vnl_y, Vnl_a}, [3]VowelSeq{Vs_u, Vs_uy, Vs_uya}, -1, VsNil, -1, VsNil},
	{3, 0, 1, [3]VnLexiName{Vnl_u, Vnl_y, Vnl_e}, [3]VowelSeq{Vs_u, Vs_uy, Vs_uye}, -1, Vs_uyer, -1, VsNil},
	{3, 1, 1, [3]VnLexiName{Vnl_u, Vnl_y, Vnl_er}, [3]VowelSeq{Vs_u, Vs_uy, Vs_uyer}, 2, VsNil, -1, VsNil},
	{3, 1, 0, [3]VnLexiName{Vnl_u, Vnl_y, Vnl_u}, [3]VowelSeq{Vs_u, Vs_uy, Vs_uyu}, -1, VsNil, -1, VsNil},
	{3, 0, 0, [3]VnLexiName{Vnl_uh, Vnl_o, Vnl_i}, [3]VowelSeq{Vs_uh, Vs_uho, Vs_uhoi}, -1, VsNil, 0, Vs_uhohi},
	{3, 0, 0, [3]VnLexiName{Vnl_uh, Vnl_o, Vnl_u}, [3]VowelSeq{Vs_uh, Vs_uho, Vs_uhou}, -1, VsNil, 0, Vs_uhohu},
	{3, 1, 0, [3]VnLexiName{Vnl_uh, Vnl_oh, Vnl_i}, [3]VowelSeq{Vs_uh, Vs_uhoh, Vs_uhohi}, -1, VsNil, 0, VsNil},
	{3, 1, 0, [3]VnLexiName{Vnl_uh, Vnl_oh, Vnl_u}, [3]VowelSeq{Vs_uh, Vs_uhoh, Vs_uhohu}, -1, VsNil, 0, VsNil},
	{3, 0, 0, [3]VnLexiName{Vnl_y, Vnl_e, Vnl_u}, [3]VowelSeq{Vs_y, Vs_ye, Vs_yeu}, -1, Vs_yeru, -1, VsNil},
	{3, 1, 0, [3]VnLexiName{Vnl_y, Vnl_er, Vnl_u}, [3]VowelSeq{Vs_y, Vs_yer, Vs_yeru}, 1, VsNil, -1, VsNil},
}

type conSeqInfo struct {
	length int
	c      [3]VnLexiName
	suffix bool
}

var cSeqList = []conSeqInfo{
	{1, [3]VnLexiName{Vnl_b, VnlNonVnChar, VnlNonVnChar}, false},
	{1, [3]VnLexiName{Vnl_c, VnlNonVnChar, VnlNonVnChar}, true},
	{2, [3]VnLexiName{Vnl_c, Vnl_h, VnlNonVnChar}, true},
	{1, [3]VnLexiName{Vnl_d, VnlNonVnChar, VnlNonVnChar}, false},
	{1, [3]VnLexiName{Vnl_dd, VnlNonVnChar, VnlNonVnChar}, false},
	{2, [3]VnLexiName{Vnl_d, Vnl_z, VnlNonVnChar}, false},
	{1, [3]VnLexiName{Vnl_g, VnlNonVnChar, VnlNonVnChar}, false},
	{2, [3]VnLexiName{Vnl_g, Vnl_h, VnlNonVnChar}, false},
	{2, [3]VnLexiName{Vnl_g, Vnl_i, VnlNonVnChar}, false},
	{3, [3]VnLexiName{Vnl_g, Vnl_i, Vnl_n}, false},
	{1, [3]VnLexiName{Vnl_k, VnlNonVnChar, VnlNonVnChar}, false},
	{2, [3]VnLexiName{Vnl_k, Vnl_h, VnlNonVnChar}, false},
	{1, [3]VnLexiName{Vnl_l, VnlNonVnChar, VnlNonVnChar}, false},
	{1, [3]VnLexiName{Vnl_m, VnlNonVnChar, VnlNonVnChar}, true},
	{1, [3]VnLexiName{Vnl_n, VnlNonVnChar, VnlNonVnChar}, true},
	{2, [3]VnLexiName{Vnl_n, Vnl_g, VnlNonVnChar}, true},
	{3, [3]VnLexiName{Vnl_n, Vnl_g, Vnl_h}, false},
	{2, [3]VnLexiName{Vnl_n, Vnl_h, VnlNonVnChar}, true},
	{1, [3]VnLexiName{Vnl_p, VnlNonVnChar, VnlNonVnChar}, true},
	{2, [3]VnLexiName{Vnl_p, Vnl_h, VnlNonVnChar}, false},
	{1, [3]VnLexiName{Vnl_q, VnlNonVnChar, VnlNonVnChar}, false},
	{2, [3]VnLexiName{Vnl_q, Vnl_u, VnlNonVnChar}, false},
	{1, [3]VnLexiName{Vnl_r, VnlNonVnChar, VnlNonVnChar}, false},
	{1, [3]VnLexiName{Vnl_s, VnlNonVnChar, VnlNonVnChar}, false},
	{1, [3]VnLexiName{Vnl_t, VnlNonVnChar, VnlNonVnChar}, true},
	{2, [3]VnLexiName{Vnl_t, Vnl_h, VnlNonVnChar}, false},
	{2, [3]VnLexiName{Vnl_t, Vnl_r, VnlNonVnChar}, false},
	{1, [3]VnLexiName{Vnl_v, VnlNonVnChar, VnlNonVnChar}, false},
	{1, [3]VnLexiName{Vnl_x, VnlNonVnChar, VnlNonVnChar}, false},
}

type vcPair struct {
	v VowelSeq
	c ConSeq
}

var vcPairList = []vcPair{
	{Vs_a, CsC}, {Vs_a, CsCh}, {Vs_a, CsM}, {Vs_a, CsN}, {Vs_a, CsNg},
	{Vs_a, CsNh}, {Vs_a, CsP}, {Vs_a, CsT},
	{Vs_ar, CsC}, {Vs_ar, CsM}, {Vs_ar, CsN}, {Vs_ar, CsNg}, {Vs_ar, CsP}, {Vs_ar, CsT},
	{Vs_ab, CsC}, {Vs_ab, CsM}, {Vs_ab, CsN}, {Vs_ab, CsNg}, {Vs_ab, CsP}, {Vs_ab, CsT},

	{Vs_e, CsC}, {Vs_e, CsCh}, {Vs_e, CsM}, {Vs_e, CsN}, {Vs_e, CsNg},
	{Vs_e, CsNh}, {Vs_e, CsP}, {Vs_e, CsT},
	{Vs_er, CsC}, {Vs_er, CsCh}, {Vs_er, CsM}, {Vs_er, CsN}, {Vs_er, CsNh},
	{Vs_er, CsP}, {Vs_er, CsT},

	{Vs_i, CsC}, {Vs_i, CsCh}, {Vs_i, CsM}, {Vs_i, CsN}, {Vs_i, CsNh}, {Vs_i, CsP}, {Vs_i, CsT},

	{Vs_o, CsC}, {Vs_o, CsM}, {Vs_o, CsN}, {Vs_o, CsNg}, {Vs_o, CsP}, {Vs_o, CsT},
	{Vs_or, CsC}, {Vs_or, CsM}, {Vs_or, CsN}, {Vs_or, CsNg}, {Vs_or, CsP}, {Vs_or, CsT},
	{Vs_oh, CsM}, {Vs_oh, CsN}, {Vs_oh, CsP}, {Vs_oh, CsT},

	{Vs_u, CsC}, {Vs_u, CsM}, {Vs_u, CsN}, {Vs_u, CsNg}, {Vs_u, CsP}, {Vs_u, CsT},
	{Vs_uh, CsC}, {Vs_uh, CsM}, {Vs_uh, CsN}, {Vs_uh, CsNg}, {Vs_uh, CsT},

	{Vs_y, CsT},
	{Vs_ie, CsC}, {Vs_ie, CsM}, {Vs_ie, CsN}, {Vs_ie, CsNg}, {Vs_ie, CsP}, {Vs_ie, CsT},
	{Vs_ier, CsC}, {Vs_ier, CsM}, {Vs_ier, CsN}, {Vs_ier, CsNg}, {Vs_ier, CsP}, {Vs_ier, CsT},

	{Vs_oa, CsC}, {Vs_oa, CsCh}, {Vs_oa, CsM}, {Vs_oa, CsN}, {Vs_oa, CsNg},
	{Vs_oa, CsNh}, {Vs_oa, CsP}, {Vs_oa, CsT},
	{Vs_oab, CsC}, {Vs_oab, CsM}, {Vs_oab, CsN}, {Vs_oab, CsNg}, {Vs_oab, CsT},

	{Vs_oe, CsN}, {Vs_oe, CsT},

	{Vs_ua, CsN}, {Vs_ua, CsNg}, {Vs_ua, CsT},
	{Vs_uar, CsN}, {Vs_uar, CsNg}, {Vs_uar, CsT},

	{Vs_ue, CsC}, {Vs_ue, CsCh}, {Vs_ue, CsN}, {Vs_ue, CsNh},
	{Vs_uer, CsC}, {Vs_uer, CsCh}, {Vs_uer, CsN}, {Vs_uer, CsNh},

	{Vs_uo, CsC}, {Vs_uo, CsM}, {Vs_uo, CsN}, {Vs_uo, CsNg}, {Vs_uo, CsP}, {Vs_uo, CsT},
	{Vs_uor, CsC}, {Vs_uor, CsM}, {Vs_uor, CsN}, {Vs_uor, CsNg}, {Vs_uor, CsT},
	{Vs_uho, CsC}, {Vs_uho, CsM}, {Vs_uho, CsN}, {Vs_uho, CsNg}, {Vs_uho, CsP}, {Vs_uho, CsT},
	{Vs_uhoh, CsC}, {Vs_uhoh, CsM}, {Vs_uhoh, CsN}, {Vs_uhoh, CsNg}, {Vs_uhoh, CsP}, {Vs_uhoh, CsT},

	{Vs_uy, CsC}, {Vs_uy, CsCh}, {Vs_uy, CsN}, {Vs_uy, CsNh}, {Vs_uy, CsP}, {Vs_uy, CsT},

	{Vs_ye, CsM}, {Vs_ye, CsN}, {Vs_ye, CsNg}, {Vs_ye, CsP}, {Vs_ye, CsT},
	{Vs_yer, CsM}, {Vs_yer, CsN}, {Vs_yer, CsNg}, {Vs_yer, CsT},

	{Vs_uye, CsN}, {Vs_uye, CsT},
	{Vs_uyer, CsN}, {Vs_uyer, CsT},
}

// lookup structures — maps instead of C++ sorted arrays + bsearch
var vSeqLookup map[[3]VnLexiName]VowelSeq
var cSeqLookup map[[3]VnLexiName]ConSeq
var vcPairSet map[vcPair]bool

func lookupVSeq(v1, v2, v3 VnLexiName) VowelSeq {
	if vs, ok := vSeqLookup[[3]VnLexiName{v1, v2, v3}]; ok {
		return vs
	}
	return VsNil
}

func lookupCSeq(c1, c2, c3 VnLexiName) ConSeq {
	if cs, ok := cSeqLookup[[3]VnLexiName{c1, c2, c3}]; ok {
		return cs
	}
	return CsNil
}

func isVnVowel(sym VnLexiName) bool {
	if sym < 0 || int(sym) >= len(isVnVowelTab) {
		return false
	}
	return isVnVowelTab[sym]
}

var isVnVowelTab [186]bool

func changeCase(x VnLexiName) VnLexiName {
	if x == VnlNonVnChar {
		return x
	}
	if int(x)&0x01 == 0 {
		return x + 1
	}
	return x - 1
}

func vnToLower(x VnLexiName) VnLexiName {
	if x == VnlNonVnChar {
		return x
	}
	if int(x)&0x01 == 0 { // even
		return x + 1
	}
	return x
}

// ---------- key maps (inputproc.cpp) ----------

type keyMapping struct {
	key    byte
	action int
}

var telexMethodMapping = []keyMapping{
	{'Z', VneTone0},
	{'S', VneTone1},
	{'F', VneTone2},
	{'R', VneTone3},
	{'X', VneTone4},
	{'J', VneTone5},
	{'W', VneTelexW},
	{'A', VneRoofA},
	{'E', VneRoofE},
	{'O', VneRoofO},
	{'D', VneDd},
	{'[', VneCount + int(Vnl_oh)},
	{']', VneCount + int(Vnl_uh)},
	{'{', VneCount + int(VnlOh)},
	{'}', VneCount + int(VnlUh)},
}

var simpleTelexMethodMapping = []keyMapping{
	{'Z', VneTone0},
	{'S', VneTone1},
	{'F', VneTone2},
	{'R', VneTone3},
	{'X', VneTone4},
	{'J', VneTone5},
	{'W', VneHookAll},
	{'A', VneRoofA},
	{'E', VneRoofE},
	{'O', VneRoofO},
	{'D', VneDd},
}

var vniMethodMapping = []keyMapping{
	{'0', VneTone0},
	{'1', VneTone1},
	{'2', VneTone2},
	{'3', VneTone3},
	{'4', VneTone4},
	{'5', VneTone5},
	{'6', VneRoofAll},
	{'7', VneHookUO},
	{'8', VneBowl},
	{'9', VneDd},
}

var viqrMethodMapping = []keyMapping{
	{'0', VneTone0},
	{'\'', VneTone1},
	{'`', VneTone2},
	{'?', VneTone3},
	{'~', VneTone4},
	{'.', VneTone5},
	{'^', VneRoofAll},
	{'+', VneHookUO},
	{'*', VneHookUO},
	{'(', VneBowl},
	{'D', VneDd},
	{'\\', VneEscChar},
}

var msViMethodMapping = []keyMapping{
	{'5', VneTone2},
	{'%', VneTone2},
	{'6', VneTone3},
	{'^', VneTone3},
	{'7', VneTone4},
	{'&', VneTone4},
	{'8', VneTone1},
	{'*', VneTone1},
	{'9', VneTone5},
	{'(', VneTone5},
	{'1', VneCount + int(Vnl_ab)},
	{'!', VneCount + int(VnlAb)},
	{'2', VneCount + int(Vnl_ar)},
	{'@', VneCount + int(VnlAr)},
	{'3', VneCount + int(Vnl_er)},
	{'#', VneCount + int(VnlEr)},
	{'4', VneCount + int(Vnl_or)},
	{'$', VneCount + int(VnlOr)},
	{'0', VneCount + int(Vnl_dd)},
	{')', VneCount + int(VnlDD)},
	{'[', VneCount + int(Vnl_uh)},
	{']', VneCount + int(Vnl_oh)},
	{'{', VneCount + int(VnlUh)},
	{'}', VneCount + int(VnlOh)},
}

var wordBreakSyms = []byte{
	',', ';', ':', '.', '"', '\'', '!', '?', ' ',
	'<', '>', '=', '+', '-', '*', '/', '\\',
	'_', '@', '#', '$', '%', '&', '(', ')', '{', '}', '[', ']', '|',
}

// ISO western chars that are also Vietnamese chars (inputproc.cpp AscVnLexiList;
// note the original has a duplicate 0xC2 entry — the last one (A4) wins)
var ascVnLexiList = [256]VnLexiName{
	0xC0: VnlA2, 0xC1: VnlA1, 0xC2: VnlA4,
	0xC8: VnlE2, 0xC9: VnlE1, 0xCA: VnlEr,
	0xCC: VnlI2, 0xCD: VnlI1,
	0xD2: VnlO2, 0xD3: VnlO1, 0xD4: VnlOr, 0xD5: VnlO4,
	0xD9: VnlU2, 0xDA: VnlU1, 0xDD: VnlY1,
	0xE0: Vnl_a2, 0xE1: Vnl_a1, 0xE2: Vnl_ar, 0xE3: Vnl_a4,
	0xE8: Vnl_e2, 0xE9: Vnl_e1, 0xEA: Vnl_er,
	0xEC: Vnl_i2, 0xED: Vnl_i1,
	0xF2: Vnl_o2, 0xF3: Vnl_o1, 0xF4: Vnl_or, 0xF5: Vnl_o4,
	0xF9: Vnl_u2, 0xFA: Vnl_u1, 0xFD: Vnl_y1,
}

// chars that have different code points in Unicode vs Western charsets
var specialWesternChars = []byte{
	0x80, 0x82, 0x83, 0x84, 0x85, 0x86, 0x87, 0x88,
	0x89, 0x8A, 0x8B, 0x8C, 0x8E, 0x91, 0x92, 0x93,
	0x94, 0x95, 0x96, 0x97, 0x98, 0x99, 0x9A, 0x9B,
	0x9C, 0x9E, 0x9F,
}

// A-Z -> VnLexiName (inputproc.cpp AZLexiUpper/AZLexiLower)
var azLexiUpper = [26]VnLexiName{
	VnlA, VnlB, VnlC, VnlD, VnlE, VnlF, VnlG, VnlH, VnlI, VnlJ,
	VnlK, VnlL, VnlM, VnlN, VnlO, VnlP, VnlQ, VnlR, VnlS, VnlT,
	VnlU, VnlV, VnlW, VnlX, VnlY, VnlZ,
}
var azLexiLower = [26]VnLexiName{
	Vnl_a, Vnl_b, Vnl_c, Vnl_d, Vnl_e, Vnl_f, Vnl_g, Vnl_h, Vnl_i, Vnl_j,
	Vnl_k, Vnl_l, Vnl_m, Vnl_n, Vnl_o, Vnl_p, Vnl_q, Vnl_r, Vnl_s, Vnl_t,
	Vnl_u, Vnl_v, Vnl_w, Vnl_x, Vnl_y, Vnl_z,
}

// ---------- classification maps (built in init) ----------

var ukcMap [256]int // UkCharType
var isoVnLexiMap [256]VnLexiName
var isoStdVnCharMap [256]StdVnChar
var uniToStdVn map[uint16]StdVnChar // reverse of unicodeTable

func init() {
	// SetupInputClassifierTable
	var c int
	for c = 0; c <= 32; c++ {
		ukcMap[c] = UkcReset
	}
	for c = 33; c < 256; c++ {
		ukcMap[c] = UkcNonVn
	}
	for c = 'a'; c <= 'z'; c++ {
		ukcMap[c] = UkcVn
	}
	for c = 'A'; c <= 'Z'; c++ {
		ukcMap[c] = UkcVn
	}
	for asc, l := range ascVnLexiList {
		if asc != 0 && l != 0 {
			ukcMap[asc] = UkcVn
		}
	}
	ukcMap['j'] = UkcNonVn
	ukcMap['J'] = UkcNonVn
	ukcMap['f'] = UkcNonVn
	ukcMap['F'] = UkcNonVn
	ukcMap['w'] = UkcNonVn
	ukcMap['W'] = UkcNonVn
	for _, s := range wordBreakSyms {
		ukcMap[s] = UkcWordBreak
	}

	for i := range isoVnLexiMap {
		isoVnLexiMap[i] = VnlNonVnChar
	}
	for asc, l := range ascVnLexiList {
		if asc != 0 && l != 0 {
			isoVnLexiMap[asc] = l
		}
	}
	for c := 'a'; c <= 'z'; c++ {
		isoVnLexiMap[c] = azLexiLower[c-'a']
	}
	for c := 'A'; c <= 'Z'; c++ {
		isoVnLexiMap[c] = azLexiUpper[c-'A']
	}

	// engineClassInit
	vSeqLookup = make(map[[3]VnLexiName]VowelSeq, len(vSeqList))
	for i, info := range vSeqList {
		vSeqLookup[info.v] = VowelSeq(i)
	}
	cSeqLookup = make(map[[3]VnLexiName]ConSeq, len(cSeqList))
	for i, info := range cSeqList {
		cSeqLookup[info.c] = ConSeq(i)
	}
	vcPairSet = make(map[vcPair]bool, len(vcPairList))
	for _, p := range vcPairList {
		vcPairSet[p] = true
	}

	for i := range isVnVowelTab {
		isVnVowelTab[i] = true
	}
	for ch := 'a'; ch <= 'z'; ch++ {
		if ch != 'a' && ch != 'e' && ch != 'i' && ch != 'o' && ch != 'u' && ch != 'y' {
			isVnVowelTab[azLexiLower[ch-'a']] = false
			isVnVowelTab[azLexiUpper[ch-'a']] = false
		}
	}
	isVnVowelTab[Vnl_dd] = false
	isVnVowelTab[VnlDD] = false

	// SetupUnikeyEngine
	for i := range isoStdVnCharMap {
		isoStdVnCharMap[i] = StdVnChar(i)
	}
	for i, ch := range specialWesternChars {
		isoStdVnCharMap[ch] = StdVnChar(int(VnlLastChar) + i + VnStdCharOffset)
	}
	for i := 0; i < 256; i++ {
		if lexi := isoToVnLexi(uint32(i)); lexi != VnlNonVnChar {
			isoStdVnCharMap[i] = StdVnChar(int(lexi) + VnStdCharOffset)
		}
	}

	// reverse map unicode -> StdVnChar (for macro file parsing)
	uniToStdVn = make(map[uint16]StdVnChar, TotalVnChars)
	for i, u := range unicodeTable {
		if _, exists := uniToStdVn[u]; !exists {
			uniToStdVn[u] = StdVnChar(i + VnStdCharOffset)
		}
	}
}

func isoToVnLexi(keyCode uint32) VnLexiName {
	if keyCode > 255 {
		return VnlNonVnChar
	}
	return isoVnLexiMap[keyCode]
}

func isoToStdVnChar(keyCode uint32) StdVnChar {
	if keyCode < 256 {
		return isoStdVnCharMap[keyCode]
	}
	return keyCode
}

// ---------- Unicode table (vnconv/data.cpp), indexed by StdVnChar - VnStdCharOffset ----------
var unicodeTable = [TotalVnChars]uint16{
	0x0041, 0x0061, 0x00c1, 0x00e1, 0x00c0, 0x00e0, 0x1ea2, 0x1ea3, 0x00c3, 0x00e3, 0x1ea0, 0x1ea1, //a
	0x00c2, 0x00e2, 0x1ea4, 0x1ea5, 0x1ea6, 0x1ea7, 0x1ea8, 0x1ea9, 0x1eaa, 0x1eab, 0x1eac, 0x1ead, //a^
	0x0102, 0x0103, 0x1eae, 0x1eaf, 0x1eb0, 0x1eb1, 0x1eb2, 0x1eb3, 0x1eb4, 0x1eb5, 0x1eb6, 0x1eb7, //a(
	0x0042, 0x0062, 0x0043, 0x0063, 0x0044, 0x0064, //B b C c D d
	0x0110, 0x0111, // DD, dd
	0x0045, 0x0065, 0x00c9, 0x00e9, 0x00c8, 0x00e8, 0x1eba, 0x1ebb, 0x1ebc, 0x1ebd, 0x1eb8, 0x1eb9, //e
	0x00ca, 0x00ea, 0x1ebe, 0x1ebf, 0x1ec0, 0x1ec1, 0x1ec2, 0x1ec3, 0x1ec4, 0x1ec5, 0x1ec6, 0x1ec7, //e^
	0x0046, 0x0066, 0x0047, 0x0067, 0x0048, 0x0068, // F f G g H h
	0x0049, 0x0069, 0x00cd, 0x00ed, 0x00cc, 0x00ec, 0x1ec8, 0x1ec9, 0x0128, 0x0129, 0x1eca, 0x1ecb, //i
	0x004a, 0x006a, 0x004b, 0x006b, 0x004c, 0x006c, 0x004d, 0x006d, 0x004e, 0x006e, // J j K k L l M m N n
	0x004f, 0x006f, 0x00d3, 0x00f3, 0x00d2, 0x00f2, 0x1ece, 0x1ecf, 0x00d5, 0x00f5, 0x1ecc, 0x1ecd, //o
	0x00d4, 0x00f4, 0x1ed0, 0x1ed1, 0x1ed2, 0x1ed3, 0x1ed4, 0x1ed5, 0x1ed6, 0x1ed7, 0x1ed8, 0x1ed9, //o^
	0x01a0, 0x01a1, 0x1eda, 0x1edb, 0x1edc, 0x1edd, 0x1ede, 0x1edf, 0x1ee0, 0x1ee1, 0x1ee2, 0x1ee3, //o+
	0x0050, 0x0070, 0x0051, 0x0071, 0x0052, 0x0072, 0x0053, 0x0073, 0x0054, 0x0074, //P p Q q R r S s T t
	0x0055, 0x0075, 0x00da, 0x00fa, 0x00d9, 0x00f9, 0x1ee6, 0x1ee7, 0x0168, 0x0169, 0x1ee4, 0x1ee5, //u
	0x01af, 0x01b0, 0x1ee8, 0x1ee9, 0x1eea, 0x1eeb, 0x1eec, 0x1eed, 0x1eee, 0x1eef, 0x1ef0, 0x1ef1, //u+
	0x0056, 0x0076, 0x0057, 0x0077, 0x0058, 0x0078, // V v W w X x
	0x0059, 0x0079, 0x00dd, 0x00fd, 0x1ef2, 0x1ef3, 0x1ef6, 0x1ef7, 0x1ef8, 0x1ef9, 0x1ef4, 0x1ef5, //y
	0x005a, 0x007a, // Z z
	// Symbols that have different code points in Unicode and Western charsets
	0x20AC, 0x20A1, 0x0192, 0x201E, 0x2026, 0x2020, 0x2021, 0x02C6,
	0x2030, 0x0160, 0x2039, 0x0152, 0x017D, 0x2018, 0x2019, 0x201C,
	0x201D, 0x2022, 0x2013, 0x2014, 0x02DC, 0x2122, 0x0161, 0x203A,
	0x0153, 0x017E, 0x0178,
}

var stdVnRootChar = [TotalVnChars]int{
	0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, //a [A=0]
	0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, //a^ -> a
	0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, //a( -> a
	36, 37, 38, 39, 40, 41, // bcd [D=40, d=41]
	40, 41, // DD dd [mapped to D, d]
	44, 45, 44, 45, 44, 45, 44, 45, 44, 45, 44, 45, // 3: e [E = 44]
	44, 45, 44, 45, 44, 45, 44, 45, 44, 45, 44, 45, // 4: e^ -> e
	68, 69, 70, 71, 72, 73, // fgh
	74, 75, 74, 75, 74, 75, 74, 75, 74, 75, 74, 75, // 5: i
	86, 87, 88, 89, 90, 91, 92, 93, 94, 95, //jklmn
	96, 97, 96, 97, 96, 97, 96, 97, 96, 97, 96, 97, // 6: o [o=96]
	96, 97, 96, 97, 96, 97, 96, 97, 96, 97, 96, 97, // 7: o^ -> o
	96, 97, 96, 97, 96, 97, 96, 97, 96, 97, 96, 97, // 8: o+ -> o
	132, 133, 134, 135, 136, 137, 138, 139, 140, 141, // pqrst
	142, 143, 142, 143, 142, 143, 142, 143, 142, 143, 142, 143, // 9: u [U=142]
	142, 143, 142, 143, 142, 143, 142, 143, 142, 143, 142, 143, //10: u+ -> u
	166, 167, 168, 169, 170, 171, //vwx
	172, 173, 172, 173, 172, 173, 172, 173, 172, 173, 172, 173, //11: y [Y=172]
	184, 185, // z
	186, 187, 188, 189, 190, 191, 192, 193,
	194, 195, 196, 197, 198, 199, 200, 201,
	202, 203, 204, 205, 206, 207, 208, 209,
	210, 211, 212,
}

var stdVnNoTone = [TotalVnChars]int{
	0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, //a [A=0]
	12, 13, 12, 13, 12, 13, 12, 13, 12, 13, 12, 13, //a^
	24, 25, 24, 25, 24, 25, 24, 25, 24, 25, 24, 25, //a(
	36, 37, 38, 39, 40, 41, // bcd [D=40, d=41]
	42, 43, // DD dd
	44, 45, 44, 45, 44, 45, 44, 45, 44, 45, 44, 45, // 3: e [E = 44]
	56, 57, 56, 57, 56, 57, 56, 57, 56, 57, 56, 57, // 4: e^
	68, 69, 70, 71, 72, 73, // fgh
	74, 75, 74, 75, 74, 75, 74, 75, 74, 75, 74, 75, // 5: i
	86, 87, 88, 89, 90, 91, 92, 93, 94, 95, //jklmn
	96, 97, 96, 97, 96, 97, 96, 97, 96, 97, 96, 97, // 6: o [o=96]
	108, 109, 108, 109, 108, 109, 108, 109, 108, 109, 108, 109, // 7: o^
	120, 121, 120, 121, 120, 121, 120, 121, 120, 121, 120, 121, // 8: o+
	132, 133, 134, 135, 136, 137, 138, 139, 140, 141, // pqrst
	142, 143, 142, 143, 142, 143, 142, 143, 142, 143, 142, 143, // 9: u [U=142]
	154, 155, 154, 155, 154, 155, 154, 155, 154, 155, 154, 155, //10: u+
	166, 167, 168, 169, 170, 171, //vwx
	172, 173, 172, 173, 172, 173, 172, 173, 172, 173, 172, 173, //11: y [Y=172]
	184, 185, // z
	186, 187, 188, 189, 190, 191, 192, 193,
	194, 195, 196, 197, 198, 199, 200, 201,
	202, 203, 204, 205, 206, 207, 208, 209,
	210, 211, 212,
}
