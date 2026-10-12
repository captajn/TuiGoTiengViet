package com.tuigo.engine

// ---------- static tables ported verbatim from ukengine.cpp / inputproc.cpp / vnconv/data.cpp ----------

internal class VowelSeqInfo(
    val length: Int,
    val complete: Int,
    val conSuffix: Int, // allow consonant suffix
    val v: IntArray,    // [3]VnLexiName
    val sub: IntArray,  // [3]VowelSeq
    val roofPos: Int,
    val withRoof: VowelSeq,
    val hookPos: Int,
    val withHook: VowelSeq, // hook & bowl
)

internal val vSeqList = arrayOf(
    VowelSeqInfo(1, 1, 1, intArrayOf(Vnl_a, VnlNonVnChar, VnlNonVnChar), intArrayOf(Vs_a, VsNil, VsNil), -1, Vs_ar, -1, Vs_ab),
    VowelSeqInfo(1, 1, 1, intArrayOf(Vnl_ar, VnlNonVnChar, VnlNonVnChar), intArrayOf(Vs_ar, VsNil, VsNil), 0, VsNil, -1, Vs_ab),
    VowelSeqInfo(1, 1, 1, intArrayOf(Vnl_ab, VnlNonVnChar, VnlNonVnChar), intArrayOf(Vs_ab, VsNil, VsNil), -1, Vs_ar, 0, VsNil),
    VowelSeqInfo(1, 1, 1, intArrayOf(Vnl_e, VnlNonVnChar, VnlNonVnChar), intArrayOf(Vs_e, VsNil, VsNil), -1, Vs_er, -1, VsNil),
    VowelSeqInfo(1, 1, 1, intArrayOf(Vnl_er, VnlNonVnChar, VnlNonVnChar), intArrayOf(Vs_er, VsNil, VsNil), 0, VsNil, -1, VsNil),
    VowelSeqInfo(1, 1, 1, intArrayOf(Vnl_i, VnlNonVnChar, VnlNonVnChar), intArrayOf(Vs_i, VsNil, VsNil), -1, VsNil, -1, VsNil),
    VowelSeqInfo(1, 1, 1, intArrayOf(Vnl_o, VnlNonVnChar, VnlNonVnChar), intArrayOf(Vs_o, VsNil, VsNil), -1, Vs_or, -1, Vs_oh),
    VowelSeqInfo(1, 1, 1, intArrayOf(Vnl_or, VnlNonVnChar, VnlNonVnChar), intArrayOf(Vs_or, VsNil, VsNil), 0, VsNil, -1, Vs_oh),
    VowelSeqInfo(1, 1, 1, intArrayOf(Vnl_oh, VnlNonVnChar, VnlNonVnChar), intArrayOf(Vs_oh, VsNil, VsNil), -1, Vs_or, 0, VsNil),
    VowelSeqInfo(1, 1, 1, intArrayOf(Vnl_u, VnlNonVnChar, VnlNonVnChar), intArrayOf(Vs_u, VsNil, VsNil), -1, VsNil, -1, Vs_uh),
    VowelSeqInfo(1, 1, 1, intArrayOf(Vnl_uh, VnlNonVnChar, VnlNonVnChar), intArrayOf(Vs_uh, VsNil, VsNil), -1, VsNil, 0, VsNil),
    VowelSeqInfo(1, 1, 1, intArrayOf(Vnl_y, VnlNonVnChar, VnlNonVnChar), intArrayOf(Vs_y, VsNil, VsNil), -1, VsNil, -1, VsNil),
    VowelSeqInfo(2, 1, 0, intArrayOf(Vnl_a, Vnl_i, VnlNonVnChar), intArrayOf(Vs_a, Vs_ai, VsNil), -1, VsNil, -1, VsNil),
    VowelSeqInfo(2, 1, 0, intArrayOf(Vnl_a, Vnl_o, VnlNonVnChar), intArrayOf(Vs_a, Vs_ao, VsNil), -1, VsNil, -1, VsNil),
    VowelSeqInfo(2, 1, 0, intArrayOf(Vnl_a, Vnl_u, VnlNonVnChar), intArrayOf(Vs_a, Vs_au, VsNil), -1, Vs_aru, -1, VsNil),
    VowelSeqInfo(2, 1, 0, intArrayOf(Vnl_a, Vnl_y, VnlNonVnChar), intArrayOf(Vs_a, Vs_ay, VsNil), -1, Vs_ary, -1, VsNil),
    VowelSeqInfo(2, 1, 0, intArrayOf(Vnl_ar, Vnl_u, VnlNonVnChar), intArrayOf(Vs_ar, Vs_aru, VsNil), 0, VsNil, -1, VsNil),
    VowelSeqInfo(2, 1, 0, intArrayOf(Vnl_ar, Vnl_y, VnlNonVnChar), intArrayOf(Vs_ar, Vs_ary, VsNil), 0, VsNil, -1, VsNil),
    VowelSeqInfo(2, 1, 0, intArrayOf(Vnl_e, Vnl_o, VnlNonVnChar), intArrayOf(Vs_e, Vs_eo, VsNil), -1, VsNil, -1, VsNil),
    VowelSeqInfo(2, 0, 0, intArrayOf(Vnl_e, Vnl_u, VnlNonVnChar), intArrayOf(Vs_e, Vs_eu, VsNil), -1, Vs_eru, -1, VsNil),
    VowelSeqInfo(2, 1, 0, intArrayOf(Vnl_er, Vnl_u, VnlNonVnChar), intArrayOf(Vs_er, Vs_eru, VsNil), 0, VsNil, -1, VsNil),
    VowelSeqInfo(2, 1, 1, intArrayOf(Vnl_i, Vnl_a, VnlNonVnChar), intArrayOf(Vs_i, Vs_ia, VsNil), -1, VsNil, -1, VsNil),
    VowelSeqInfo(2, 0, 1, intArrayOf(Vnl_i, Vnl_e, VnlNonVnChar), intArrayOf(Vs_i, Vs_ie, VsNil), -1, Vs_ier, -1, VsNil),
    VowelSeqInfo(2, 1, 1, intArrayOf(Vnl_i, Vnl_er, VnlNonVnChar), intArrayOf(Vs_i, Vs_ier, VsNil), 1, VsNil, -1, VsNil),
    VowelSeqInfo(2, 1, 0, intArrayOf(Vnl_i, Vnl_u, VnlNonVnChar), intArrayOf(Vs_i, Vs_iu, VsNil), -1, VsNil, -1, VsNil),
    VowelSeqInfo(2, 1, 1, intArrayOf(Vnl_o, Vnl_a, VnlNonVnChar), intArrayOf(Vs_o, Vs_oa, VsNil), -1, VsNil, -1, Vs_oab),
    VowelSeqInfo(2, 1, 1, intArrayOf(Vnl_o, Vnl_ab, VnlNonVnChar), intArrayOf(Vs_o, Vs_oab, VsNil), -1, VsNil, 1, VsNil),
    VowelSeqInfo(2, 1, 1, intArrayOf(Vnl_o, Vnl_e, VnlNonVnChar), intArrayOf(Vs_o, Vs_oe, VsNil), -1, VsNil, -1, VsNil),
    VowelSeqInfo(2, 1, 0, intArrayOf(Vnl_o, Vnl_i, VnlNonVnChar), intArrayOf(Vs_o, Vs_oi, VsNil), -1, Vs_ori, -1, Vs_ohi),
    VowelSeqInfo(2, 1, 0, intArrayOf(Vnl_or, Vnl_i, VnlNonVnChar), intArrayOf(Vs_or, Vs_ori, VsNil), 0, VsNil, -1, Vs_ohi),
    VowelSeqInfo(2, 1, 0, intArrayOf(Vnl_oh, Vnl_i, VnlNonVnChar), intArrayOf(Vs_oh, Vs_ohi, VsNil), -1, Vs_ori, 0, VsNil),
    VowelSeqInfo(2, 1, 1, intArrayOf(Vnl_u, Vnl_a, VnlNonVnChar), intArrayOf(Vs_u, Vs_ua, VsNil), -1, Vs_uar, -1, Vs_uha),
    VowelSeqInfo(2, 1, 1, intArrayOf(Vnl_u, Vnl_ar, VnlNonVnChar), intArrayOf(Vs_u, Vs_uar, VsNil), 1, VsNil, -1, VsNil),
    VowelSeqInfo(2, 0, 1, intArrayOf(Vnl_u, Vnl_e, VnlNonVnChar), intArrayOf(Vs_u, Vs_ue, VsNil), -1, Vs_uer, -1, VsNil),
    VowelSeqInfo(2, 1, 1, intArrayOf(Vnl_u, Vnl_er, VnlNonVnChar), intArrayOf(Vs_u, Vs_uer, VsNil), 1, VsNil, -1, VsNil),
    VowelSeqInfo(2, 1, 0, intArrayOf(Vnl_u, Vnl_i, VnlNonVnChar), intArrayOf(Vs_u, Vs_ui, VsNil), -1, VsNil, -1, Vs_uhi),
    VowelSeqInfo(2, 0, 1, intArrayOf(Vnl_u, Vnl_o, VnlNonVnChar), intArrayOf(Vs_u, Vs_uo, VsNil), -1, Vs_uor, -1, Vs_uho),
    VowelSeqInfo(2, 1, 1, intArrayOf(Vnl_u, Vnl_or, VnlNonVnChar), intArrayOf(Vs_u, Vs_uor, VsNil), 1, VsNil, -1, Vs_uoh),
    VowelSeqInfo(2, 1, 1, intArrayOf(Vnl_u, Vnl_oh, VnlNonVnChar), intArrayOf(Vs_u, Vs_uoh, VsNil), -1, Vs_uor, 1, Vs_uhoh),
    VowelSeqInfo(2, 0, 0, intArrayOf(Vnl_u, Vnl_u, VnlNonVnChar), intArrayOf(Vs_u, Vs_uu, VsNil), -1, VsNil, -1, Vs_uhu),
    VowelSeqInfo(2, 1, 1, intArrayOf(Vnl_u, Vnl_y, VnlNonVnChar), intArrayOf(Vs_u, Vs_uy, VsNil), -1, VsNil, -1, VsNil),
    VowelSeqInfo(2, 1, 0, intArrayOf(Vnl_uh, Vnl_a, VnlNonVnChar), intArrayOf(Vs_uh, Vs_uha, VsNil), -1, VsNil, 0, VsNil),
    VowelSeqInfo(2, 1, 0, intArrayOf(Vnl_uh, Vnl_i, VnlNonVnChar), intArrayOf(Vs_uh, Vs_uhi, VsNil), -1, VsNil, 0, VsNil),
    VowelSeqInfo(2, 0, 1, intArrayOf(Vnl_uh, Vnl_o, VnlNonVnChar), intArrayOf(Vs_uh, Vs_uho, VsNil), -1, VsNil, 0, Vs_uhoh),
    VowelSeqInfo(2, 1, 1, intArrayOf(Vnl_uh, Vnl_oh, VnlNonVnChar), intArrayOf(Vs_uh, Vs_uhoh, VsNil), -1, VsNil, 0, VsNil),
    VowelSeqInfo(2, 1, 0, intArrayOf(Vnl_uh, Vnl_u, VnlNonVnChar), intArrayOf(Vs_uh, Vs_uhu, VsNil), -1, VsNil, 0, VsNil),
    VowelSeqInfo(2, 0, 1, intArrayOf(Vnl_y, Vnl_e, VnlNonVnChar), intArrayOf(Vs_y, Vs_ye, VsNil), -1, Vs_yer, -1, VsNil),
    VowelSeqInfo(2, 1, 1, intArrayOf(Vnl_y, Vnl_er, VnlNonVnChar), intArrayOf(Vs_y, Vs_yer, VsNil), 1, VsNil, -1, VsNil),
    VowelSeqInfo(3, 0, 0, intArrayOf(Vnl_i, Vnl_e, Vnl_u), intArrayOf(Vs_i, Vs_ie, Vs_ieu), -1, Vs_ieru, -1, VsNil),
    VowelSeqInfo(3, 1, 0, intArrayOf(Vnl_i, Vnl_er, Vnl_u), intArrayOf(Vs_i, Vs_ier, Vs_ieru), 1, VsNil, -1, VsNil),
    VowelSeqInfo(3, 1, 0, intArrayOf(Vnl_o, Vnl_a, Vnl_i), intArrayOf(Vs_o, Vs_oa, Vs_oai), -1, VsNil, -1, VsNil),
    VowelSeqInfo(3, 1, 0, intArrayOf(Vnl_o, Vnl_a, Vnl_y), intArrayOf(Vs_o, Vs_oa, Vs_oay), -1, VsNil, -1, VsNil),
    VowelSeqInfo(3, 1, 0, intArrayOf(Vnl_o, Vnl_e, Vnl_o), intArrayOf(Vs_o, Vs_oe, Vs_oeo), -1, VsNil, -1, VsNil),
    VowelSeqInfo(3, 0, 0, intArrayOf(Vnl_u, Vnl_a, Vnl_y), intArrayOf(Vs_u, Vs_ua, Vs_uay), -1, Vs_uary, -1, VsNil),
    VowelSeqInfo(3, 1, 0, intArrayOf(Vnl_u, Vnl_ar, Vnl_y), intArrayOf(Vs_u, Vs_uar, Vs_uary), 1, VsNil, -1, VsNil),
    VowelSeqInfo(3, 0, 0, intArrayOf(Vnl_u, Vnl_o, Vnl_i), intArrayOf(Vs_u, Vs_uo, Vs_uoi), -1, Vs_uori, -1, Vs_uhoi),
    VowelSeqInfo(3, 0, 0, intArrayOf(Vnl_u, Vnl_o, Vnl_u), intArrayOf(Vs_u, Vs_uo, Vs_uou), -1, VsNil, -1, Vs_uhou),
    VowelSeqInfo(3, 1, 0, intArrayOf(Vnl_u, Vnl_or, Vnl_i), intArrayOf(Vs_u, Vs_uor, Vs_uori), 1, VsNil, -1, Vs_uohi),
    VowelSeqInfo(3, 0, 0, intArrayOf(Vnl_u, Vnl_oh, Vnl_i), intArrayOf(Vs_u, Vs_uoh, Vs_uohi), -1, Vs_uori, 1, Vs_uhohi),
    VowelSeqInfo(3, 0, 0, intArrayOf(Vnl_u, Vnl_oh, Vnl_u), intArrayOf(Vs_u, Vs_uoh, Vs_uohu), -1, VsNil, 1, Vs_uhohu),
    VowelSeqInfo(3, 1, 0, intArrayOf(Vnl_u, Vnl_y, Vnl_a), intArrayOf(Vs_u, Vs_uy, Vs_uya), -1, VsNil, -1, VsNil),
    VowelSeqInfo(3, 0, 1, intArrayOf(Vnl_u, Vnl_y, Vnl_e), intArrayOf(Vs_u, Vs_uy, Vs_uye), -1, Vs_uyer, -1, VsNil),
    VowelSeqInfo(3, 1, 1, intArrayOf(Vnl_u, Vnl_y, Vnl_er), intArrayOf(Vs_u, Vs_uy, Vs_uyer), 2, VsNil, -1, VsNil),
    VowelSeqInfo(3, 1, 0, intArrayOf(Vnl_u, Vnl_y, Vnl_u), intArrayOf(Vs_u, Vs_uy, Vs_uyu), -1, VsNil, -1, VsNil),
    VowelSeqInfo(3, 0, 0, intArrayOf(Vnl_uh, Vnl_o, Vnl_i), intArrayOf(Vs_uh, Vs_uho, Vs_uhoi), -1, VsNil, 0, Vs_uhohi),
    VowelSeqInfo(3, 0, 0, intArrayOf(Vnl_uh, Vnl_o, Vnl_u), intArrayOf(Vs_uh, Vs_uho, Vs_uhou), -1, VsNil, 0, Vs_uhohu),
    VowelSeqInfo(3, 1, 0, intArrayOf(Vnl_uh, Vnl_oh, Vnl_i), intArrayOf(Vs_uh, Vs_uhoh, Vs_uhohi), -1, VsNil, 0, VsNil),
    VowelSeqInfo(3, 1, 0, intArrayOf(Vnl_uh, Vnl_oh, Vnl_u), intArrayOf(Vs_uh, Vs_uhoh, Vs_uhohu), -1, VsNil, 0, VsNil),
    VowelSeqInfo(3, 0, 0, intArrayOf(Vnl_y, Vnl_e, Vnl_u), intArrayOf(Vs_y, Vs_ye, Vs_yeu), -1, Vs_yeru, -1, VsNil),
    VowelSeqInfo(3, 1, 0, intArrayOf(Vnl_y, Vnl_er, Vnl_u), intArrayOf(Vs_y, Vs_yer, Vs_yeru), 1, VsNil, -1, VsNil),
)

internal class ConSeqInfo(val length: Int, val c: IntArray, val suffix: Boolean)

internal val cSeqList = arrayOf(
    ConSeqInfo(1, intArrayOf(Vnl_b, VnlNonVnChar, VnlNonVnChar), false),
    ConSeqInfo(1, intArrayOf(Vnl_c, VnlNonVnChar, VnlNonVnChar), true),
    ConSeqInfo(2, intArrayOf(Vnl_c, Vnl_h, VnlNonVnChar), true),
    ConSeqInfo(1, intArrayOf(Vnl_d, VnlNonVnChar, VnlNonVnChar), false),
    ConSeqInfo(1, intArrayOf(Vnl_dd, VnlNonVnChar, VnlNonVnChar), false),
    ConSeqInfo(2, intArrayOf(Vnl_d, Vnl_z, VnlNonVnChar), false),
    ConSeqInfo(1, intArrayOf(Vnl_g, VnlNonVnChar, VnlNonVnChar), false),
    ConSeqInfo(2, intArrayOf(Vnl_g, Vnl_h, VnlNonVnChar), false),
    ConSeqInfo(2, intArrayOf(Vnl_g, Vnl_i, VnlNonVnChar), false),
    ConSeqInfo(3, intArrayOf(Vnl_g, Vnl_i, Vnl_n), false),
    ConSeqInfo(1, intArrayOf(Vnl_k, VnlNonVnChar, VnlNonVnChar), false),
    ConSeqInfo(2, intArrayOf(Vnl_k, Vnl_h, VnlNonVnChar), false),
    ConSeqInfo(1, intArrayOf(Vnl_l, VnlNonVnChar, VnlNonVnChar), false),
    ConSeqInfo(1, intArrayOf(Vnl_m, VnlNonVnChar, VnlNonVnChar), true),
    ConSeqInfo(1, intArrayOf(Vnl_n, VnlNonVnChar, VnlNonVnChar), true),
    ConSeqInfo(2, intArrayOf(Vnl_n, Vnl_g, VnlNonVnChar), true),
    ConSeqInfo(3, intArrayOf(Vnl_n, Vnl_g, Vnl_h), false),
    ConSeqInfo(2, intArrayOf(Vnl_n, Vnl_h, VnlNonVnChar), true),
    ConSeqInfo(1, intArrayOf(Vnl_p, VnlNonVnChar, VnlNonVnChar), true),
    ConSeqInfo(2, intArrayOf(Vnl_p, Vnl_h, VnlNonVnChar), false),
    ConSeqInfo(1, intArrayOf(Vnl_q, VnlNonVnChar, VnlNonVnChar), false),
    ConSeqInfo(2, intArrayOf(Vnl_q, Vnl_u, VnlNonVnChar), false),
    ConSeqInfo(1, intArrayOf(Vnl_r, VnlNonVnChar, VnlNonVnChar), false),
    ConSeqInfo(1, intArrayOf(Vnl_s, VnlNonVnChar, VnlNonVnChar), false),
    ConSeqInfo(1, intArrayOf(Vnl_t, VnlNonVnChar, VnlNonVnChar), true),
    ConSeqInfo(2, intArrayOf(Vnl_t, Vnl_h, VnlNonVnChar), false),
    ConSeqInfo(2, intArrayOf(Vnl_t, Vnl_r, VnlNonVnChar), false),
    ConSeqInfo(1, intArrayOf(Vnl_v, VnlNonVnChar, VnlNonVnChar), false),
    ConSeqInfo(1, intArrayOf(Vnl_x, VnlNonVnChar, VnlNonVnChar), false),
)

// (vowel seq, consonant seq) pairs; packed into Int set: v*64+c
private val vcPairList = intArrayOf(
    Vs_a * 64 + CsC, Vs_a * 64 + CsCh, Vs_a * 64 + CsM, Vs_a * 64 + CsN, Vs_a * 64 + CsNg,
    Vs_a * 64 + CsNh, Vs_a * 64 + CsP, Vs_a * 64 + CsT,
    Vs_ar * 64 + CsC, Vs_ar * 64 + CsM, Vs_ar * 64 + CsN, Vs_ar * 64 + CsNg, Vs_ar * 64 + CsP, Vs_ar * 64 + CsT,
    Vs_ab * 64 + CsC, Vs_ab * 64 + CsM, Vs_ab * 64 + CsN, Vs_ab * 64 + CsNg, Vs_ab * 64 + CsP, Vs_ab * 64 + CsT,

    Vs_e * 64 + CsC, Vs_e * 64 + CsCh, Vs_e * 64 + CsM, Vs_e * 64 + CsN, Vs_e * 64 + CsNg,
    Vs_e * 64 + CsNh, Vs_e * 64 + CsP, Vs_e * 64 + CsT,
    Vs_er * 64 + CsC, Vs_er * 64 + CsCh, Vs_er * 64 + CsM, Vs_er * 64 + CsN, Vs_er * 64 + CsNh,
    Vs_er * 64 + CsP, Vs_er * 64 + CsT,

    Vs_i * 64 + CsC, Vs_i * 64 + CsCh, Vs_i * 64 + CsM, Vs_i * 64 + CsN, Vs_i * 64 + CsNh, Vs_i * 64 + CsP, Vs_i * 64 + CsT,

    Vs_o * 64 + CsC, Vs_o * 64 + CsM, Vs_o * 64 + CsN, Vs_o * 64 + CsNg, Vs_o * 64 + CsP, Vs_o * 64 + CsT,
    Vs_or * 64 + CsC, Vs_or * 64 + CsM, Vs_or * 64 + CsN, Vs_or * 64 + CsNg, Vs_or * 64 + CsP, Vs_or * 64 + CsT,
    Vs_oh * 64 + CsM, Vs_oh * 64 + CsN, Vs_oh * 64 + CsP, Vs_oh * 64 + CsT,

    Vs_u * 64 + CsC, Vs_u * 64 + CsM, Vs_u * 64 + CsN, Vs_u * 64 + CsNg, Vs_u * 64 + CsP, Vs_u * 64 + CsT,
    Vs_uh * 64 + CsC, Vs_uh * 64 + CsM, Vs_uh * 64 + CsN, Vs_uh * 64 + CsNg, Vs_uh * 64 + CsT,

    Vs_y * 64 + CsT,
    Vs_ie * 64 + CsC, Vs_ie * 64 + CsM, Vs_ie * 64 + CsN, Vs_ie * 64 + CsNg, Vs_ie * 64 + CsP, Vs_ie * 64 + CsT,
    Vs_ier * 64 + CsC, Vs_ier * 64 + CsM, Vs_ier * 64 + CsN, Vs_ier * 64 + CsNg, Vs_ier * 64 + CsP, Vs_ier * 64 + CsT,

    Vs_oa * 64 + CsC, Vs_oa * 64 + CsCh, Vs_oa * 64 + CsM, Vs_oa * 64 + CsN, Vs_oa * 64 + CsNg,
    Vs_oa * 64 + CsNh, Vs_oa * 64 + CsP, Vs_oa * 64 + CsT,
    Vs_oab * 64 + CsC, Vs_oab * 64 + CsM, Vs_oab * 64 + CsN, Vs_oab * 64 + CsNg, Vs_oab * 64 + CsT,

    Vs_oe * 64 + CsN, Vs_oe * 64 + CsT,

    Vs_ua * 64 + CsN, Vs_ua * 64 + CsNg, Vs_ua * 64 + CsT,
    Vs_uar * 64 + CsN, Vs_uar * 64 + CsNg, Vs_uar * 64 + CsT,

    Vs_ue * 64 + CsC, Vs_ue * 64 + CsCh, Vs_ue * 64 + CsN, Vs_ue * 64 + CsNh,
    Vs_uer * 64 + CsC, Vs_uer * 64 + CsCh, Vs_uer * 64 + CsN, Vs_uer * 64 + CsNh,

    Vs_uo * 64 + CsC, Vs_uo * 64 + CsM, Vs_uo * 64 + CsN, Vs_uo * 64 + CsNg, Vs_uo * 64 + CsP, Vs_uo * 64 + CsT,
    Vs_uor * 64 + CsC, Vs_uor * 64 + CsM, Vs_uor * 64 + CsN, Vs_uor * 64 + CsNg, Vs_uor * 64 + CsP, Vs_uor * 64 + CsT,
    Vs_uho * 64 + CsC, Vs_uho * 64 + CsM, Vs_uho * 64 + CsN, Vs_uho * 64 + CsNg, Vs_uho * 64 + CsP, Vs_uho * 64 + CsT,
    Vs_uhoh * 64 + CsC, Vs_uhoh * 64 + CsM, Vs_uhoh * 64 + CsN, Vs_uhoh * 64 + CsNg, Vs_uhoh * 64 + CsP, Vs_uhoh * 64 + CsT,

    Vs_uy * 64 + CsC, Vs_uy * 64 + CsCh, Vs_uy * 64 + CsN, Vs_uy * 64 + CsNh, Vs_uy * 64 + CsP, Vs_uy * 64 + CsT,

    Vs_ye * 64 + CsM, Vs_ye * 64 + CsN, Vs_ye * 64 + CsNg, Vs_ye * 64 + CsP, Vs_ye * 64 + CsT,
    Vs_yer * 64 + CsM, Vs_yer * 64 + CsN, Vs_yer * 64 + CsNg, Vs_yer * 64 + CsT,

    Vs_uye * 64 + CsN, Vs_uye * 64 + CsT,
    Vs_uyer * 64 + CsN, Vs_uyer * 64 + CsT,
)

internal val vcPairSet: Set<Int> = vcPairList.toHashSet()

// lookup structures — packed key: (a+1)*65536 + (b+1)*256 + (c+1)  (VnlNonVnChar=-1 -> 0)
private fun pack3(a: Int, b: Int, c: Int) = (a + 1) * 65536 + (b + 1) * 256 + (c + 1)

internal val vSeqLookup: Map<Int, VowelSeq> =
    HashMap<Int, VowelSeq>(vSeqList.size * 2).apply {
        for (i in vSeqList.indices) {
            val v = vSeqList[i].v
            put(pack3(v[0], v[1], v[2]), i)
        }
    }

internal val cSeqLookup: Map<Int, ConSeq> =
    HashMap<Int, ConSeq>(cSeqList.size * 2).apply {
        for (i in cSeqList.indices) {
            val c = cSeqList[i].c
            put(pack3(c[0], c[1], c[2]), i)
        }
    }

internal fun lookupVSeq(v1: VnLexiName, v2: VnLexiName, v3: VnLexiName): VowelSeq =
    vSeqLookup[pack3(v1, v2, v3)] ?: VsNil

internal fun lookupCSeq(c1: VnLexiName, c2: VnLexiName, c3: VnLexiName): ConSeq =
    cSeqLookup[pack3(c1, c2, c3)] ?: CsNil

internal fun isVnVowel(sym: VnLexiName): Boolean =
    sym >= 0 && sym < isVnVowelTab.size && isVnVowelTab[sym]

// A-Z -> VnLexiName (inputproc.cpp AZLexiUpper/AZLexiLower)
internal val azLexiUpper = intArrayOf(
    VnlA, VnlB, VnlC, VnlD, VnlE, VnlF, VnlG, VnlH, VnlI, VnlJ,
    VnlK, VnlL, VnlM, VnlN, VnlO, VnlP, VnlQ, VnlR, VnlS, VnlT,
    VnlU, VnlV, VnlW, VnlX, VnlY, VnlZ,
)
internal val azLexiLower = intArrayOf(
    Vnl_a, Vnl_b, Vnl_c, Vnl_d, Vnl_e, Vnl_f, Vnl_g, Vnl_h, Vnl_i, Vnl_j,
    Vnl_k, Vnl_l, Vnl_m, Vnl_n, Vnl_o, Vnl_p, Vnl_q, Vnl_r, Vnl_s, Vnl_t,
    Vnl_u, Vnl_v, Vnl_w, Vnl_x, Vnl_y, Vnl_z,
)

internal val isVnVowelTab = BooleanArray(186) { true }.apply {
    for (ch in 'a'..'z') {
        if (ch != 'a' && ch != 'e' && ch != 'i' && ch != 'o' && ch != 'u' && ch != 'y') {
            this[azLexiLower[ch - 'a']] = false
            this[azLexiUpper[ch - 'a']] = false
        }
    }
    this[Vnl_dd] = false
    this[VnlDD] = false
}

internal fun changeCase(x: VnLexiName): VnLexiName {
    if (x == VnlNonVnChar) return x
    return if (x and 0x01 == 0) x + 1 else x - 1
}

internal fun vnToLower(x: VnLexiName): VnLexiName {
    if (x == VnlNonVnChar) return x
    return if (x and 0x01 == 0) x + 1 else x // even -> lower
}

// ---------- key maps (inputproc.cpp) ----------

internal class KeyMapping(val key: Int, val action: Int)

internal val telexMethodMapping = arrayOf(
    KeyMapping('Z'.code, VneTone0),
    KeyMapping('S'.code, VneTone1),
    KeyMapping('F'.code, VneTone2),
    KeyMapping('R'.code, VneTone3),
    KeyMapping('X'.code, VneTone4),
    KeyMapping('J'.code, VneTone5),
    KeyMapping('W'.code, VneTelexW),
    KeyMapping('A'.code, VneRoofA),
    KeyMapping('E'.code, VneRoofE),
    KeyMapping('O'.code, VneRoofO),
    KeyMapping('D'.code, VneDd),
    KeyMapping('['.code, VneCount + Vnl_oh),
    KeyMapping(']'.code, VneCount + Vnl_uh),
    KeyMapping('{'.code, VneCount + VnlOh),
    KeyMapping('}'.code, VneCount + VnlUh),
)

internal val simpleTelexMethodMapping = arrayOf(
    KeyMapping('Z'.code, VneTone0),
    KeyMapping('S'.code, VneTone1),
    KeyMapping('F'.code, VneTone2),
    KeyMapping('R'.code, VneTone3),
    KeyMapping('X'.code, VneTone4),
    KeyMapping('J'.code, VneTone5),
    KeyMapping('W'.code, VneHookAll),
    KeyMapping('A'.code, VneRoofA),
    KeyMapping('E'.code, VneRoofE),
    KeyMapping('O'.code, VneRoofO),
    KeyMapping('D'.code, VneDd),
)

internal val vniMethodMapping = arrayOf(
    KeyMapping('0'.code, VneTone0),
    KeyMapping('1'.code, VneTone1),
    KeyMapping('2'.code, VneTone2),
    KeyMapping('3'.code, VneTone3),
    KeyMapping('4'.code, VneTone4),
    KeyMapping('5'.code, VneTone5),
    KeyMapping('6'.code, VneRoofAll),
    KeyMapping('7'.code, VneHookUO),
    KeyMapping('8'.code, VneBowl),
    KeyMapping('9'.code, VneDd),
)

internal val viqrMethodMapping = arrayOf(
    KeyMapping('0'.code, VneTone0),
    KeyMapping('\''.code, VneTone1),
    KeyMapping('`'.code, VneTone2),
    KeyMapping('?'.code, VneTone3),
    KeyMapping('~'.code, VneTone4),
    KeyMapping('.'.code, VneTone5),
    KeyMapping('^'.code, VneRoofAll),
    KeyMapping('+'.code, VneHookUO),
    KeyMapping('*'.code, VneHookUO),
    KeyMapping('('.code, VneBowl),
    KeyMapping('D'.code, VneDd),
    KeyMapping('\\'.code, VneEscChar),
)

internal val msViMethodMapping = arrayOf(
    KeyMapping('5'.code, VneTone2),
    KeyMapping('%'.code, VneTone2),
    KeyMapping('6'.code, VneTone3),
    KeyMapping('^'.code, VneTone3),
    KeyMapping('7'.code, VneTone4),
    KeyMapping('&'.code, VneTone4),
    KeyMapping('8'.code, VneTone1),
    KeyMapping('*'.code, VneTone1),
    KeyMapping('9'.code, VneTone5),
    KeyMapping('('.code, VneTone5),
    KeyMapping('1'.code, VneCount + Vnl_ab),
    KeyMapping('!'.code, VneCount + VnlAb),
    KeyMapping('2'.code, VneCount + Vnl_ar),
    KeyMapping('@'.code, VneCount + VnlAr),
    KeyMapping('3'.code, VneCount + Vnl_er),
    KeyMapping('#'.code, VneCount + VnlEr),
    KeyMapping('4'.code, VneCount + Vnl_or),
    KeyMapping('$'.code, VneCount + VnlOr),
    KeyMapping('0'.code, VneCount + Vnl_dd),
    KeyMapping(')'.code, VneCount + VnlDD),
    KeyMapping('['.code, VneCount + Vnl_uh),
    KeyMapping(']'.code, VneCount + Vnl_oh),
    KeyMapping('{'.code, VneCount + VnlUh),
    KeyMapping('}'.code, VneCount + VnlOh),
)

internal val wordBreakSyms = intArrayOf(
    ','.code, ';'.code, ':'.code, '.'.code, '"'.code, '\''.code, '!'.code, '?'.code, ' '.code,
    '<'.code, '>'.code, '='.code, '+'.code, '-'.code, '*'.code, '/'.code, '\\'.code,
    '_'.code, '@'.code, '#'.code, '$'.code, '%'.code, '&'.code, '('.code, ')'.code, '{'.code, '}'.code, '['.code, ']'.code, '|'.code,
)

// ISO western chars that are also Vietnamese chars (inputproc.cpp AscVnLexiList;
// note the original has a duplicate 0xC2 entry — the last one (A4) wins)
private val ascVnLexiEntries = intArrayOf(
    0xC0, VnlA2, 0xC1, VnlA1, 0xC2, VnlA4,
    0xC8, VnlE2, 0xC9, VnlE1, 0xCA, VnlEr,
    0xCC, VnlI2, 0xCD, VnlI1,
    0xD2, VnlO2, 0xD3, VnlO1, 0xD4, VnlOr, 0xD5, VnlO4,
    0xD9, VnlU2, 0xDA, VnlU1, 0xDD, VnlY1,
    0xE0, Vnl_a2, 0xE1, Vnl_a1, 0xE2, Vnl_ar, 0xE3, Vnl_a4,
    0xE8, Vnl_e2, 0xE9, Vnl_e1, 0xEA, Vnl_er,
    0xEC, Vnl_i2, 0xED, Vnl_i1,
    0xF2, Vnl_o2, 0xF3, Vnl_o1, 0xF4, Vnl_or, 0xF5, Vnl_o4,
    0xF9, Vnl_u2, 0xFA, Vnl_u1, 0xFD, Vnl_y1,
)

// chars that have different code points in Unicode vs Western charsets
internal val specialWesternChars = intArrayOf(
    0x80, 0x82, 0x83, 0x84, 0x85, 0x86, 0x87, 0x88,
    0x89, 0x8A, 0x8B, 0x8C, 0x8E, 0x91, 0x92, 0x93,
    0x94, 0x95, 0x96, 0x97, 0x98, 0x99, 0x9A, 0x9B,
    0x9C, 0x9E, 0x9F,
)

// ---------- classification maps (Go init()) ----------

internal val ukcMap = IntArray(256).apply { // UkCharType
    for (c in 0..32) this[c] = UkcReset
    for (c in 33 until 256) this[c] = UkcNonVn
    for (c in 'a'.code..'z'.code) this[c] = UkcVn
    for (c in 'A'.code..'Z'.code) this[c] = UkcVn
    var i = 0
    while (i < ascVnLexiEntries.size) {
        val asc = ascVnLexiEntries[i]
        if (asc != 0) this[asc] = UkcVn
        i += 2
    }
    this['j'.code] = UkcNonVn
    this['J'.code] = UkcNonVn
    this['f'.code] = UkcNonVn
    this['F'.code] = UkcNonVn
    this['w'.code] = UkcNonVn
    this['W'.code] = UkcNonVn
    for (s in wordBreakSyms) this[s] = UkcWordBreak
}

internal val isoVnLexiMap = IntArray(256) { VnlNonVnChar }.apply {
    var i = 0
    while (i < ascVnLexiEntries.size) {
        val asc = ascVnLexiEntries[i]
        if (asc != 0) this[asc] = ascVnLexiEntries[i + 1]
        i += 2
    }
    for (c in 'a'.code..'z'.code) this[c] = azLexiLower[c - 'a'.code]
    for (c in 'A'.code..'Z'.code) this[c] = azLexiUpper[c - 'A'.code]
}

internal fun isoToVnLexi(keyCode: Int): VnLexiName =
    if (keyCode > 255 || keyCode < 0) VnlNonVnChar else isoVnLexiMap[keyCode]

// SetupUnikeyEngine — must run after isoVnLexiMap
internal val isoStdVnCharMap = IntArray(256) { it }.apply {
    for (i in specialWesternChars.indices) {
        this[specialWesternChars[i]] = VnlLastChar + i + VnStdCharOffset
    }
    for (i in 0 until 256) {
        val lexi = isoToVnLexi(i)
        if (lexi != VnlNonVnChar) {
            this[i] = lexi + VnStdCharOffset
        }
    }
}

internal fun isoToStdVnChar(keyCode: Int): StdVnChar =
    if (keyCode in 0 until 256) isoStdVnCharMap[keyCode] else keyCode

// ---------- Unicode table (vnconv/data.cpp), indexed by StdVnChar - VnStdCharOffset ----------
internal val unicodeTable = intArrayOf(
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
)

internal val stdVnRootChar = intArrayOf(
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
)

internal val stdVnNoTone = intArrayOf(
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
)

// reverse map unicode -> StdVnChar (for macro file parsing)
internal val uniToStdVn: Map<Int, StdVnChar> =
    HashMap<Int, StdVnChar>(TotalVnChars * 2).apply {
        for (i in unicodeTable.indices) {
            val u = unicodeTable[i]
            if (!containsKey(u)) put(u, i + VnStdCharOffset)
        }
    }
