// Static tables ported verbatim from ukengine.cpp / inputproc.cpp / vnconv/data.cpp

extension String {
    /// ASCII/Unicode scalar value as Int (Go char-literal readability).
    var ord: Int { Int(unicodeScalars.first!.value) }
}

extension Character {
    var ord: Int { Int(unicodeScalars.first!.value) }
}

struct VowelSeqInfo {
    let length: Int
    let complete: Int
    let conSuffix: Int // allow consonant suffix
    let v: [VnLexiName] // [3]
    let sub: [VowelSeq] // [3]
    let roofPos: Int
    let withRoof: VowelSeq
    let hookPos: Int
    let withHook: VowelSeq // hook & bowl
}

let vSeqList: [VowelSeqInfo] = [
    VowelSeqInfo(length: 1, complete: 1, conSuffix: 1, v: [Vnl_a, VnlNonVnChar, VnlNonVnChar], sub: [Vs_a, VsNil, VsNil], roofPos: -1, withRoof: Vs_ar, hookPos: -1, withHook: Vs_ab),
    VowelSeqInfo(length: 1, complete: 1, conSuffix: 1, v: [Vnl_ar, VnlNonVnChar, VnlNonVnChar], sub: [Vs_ar, VsNil, VsNil], roofPos: 0, withRoof: VsNil, hookPos: -1, withHook: Vs_ab),
    VowelSeqInfo(length: 1, complete: 1, conSuffix: 1, v: [Vnl_ab, VnlNonVnChar, VnlNonVnChar], sub: [Vs_ab, VsNil, VsNil], roofPos: -1, withRoof: Vs_ar, hookPos: 0, withHook: VsNil),
    VowelSeqInfo(length: 1, complete: 1, conSuffix: 1, v: [Vnl_e, VnlNonVnChar, VnlNonVnChar], sub: [Vs_e, VsNil, VsNil], roofPos: -1, withRoof: Vs_er, hookPos: -1, withHook: VsNil),
    VowelSeqInfo(length: 1, complete: 1, conSuffix: 1, v: [Vnl_er, VnlNonVnChar, VnlNonVnChar], sub: [Vs_er, VsNil, VsNil], roofPos: 0, withRoof: VsNil, hookPos: -1, withHook: VsNil),
    VowelSeqInfo(length: 1, complete: 1, conSuffix: 1, v: [Vnl_i, VnlNonVnChar, VnlNonVnChar], sub: [Vs_i, VsNil, VsNil], roofPos: -1, withRoof: VsNil, hookPos: -1, withHook: VsNil),
    VowelSeqInfo(length: 1, complete: 1, conSuffix: 1, v: [Vnl_o, VnlNonVnChar, VnlNonVnChar], sub: [Vs_o, VsNil, VsNil], roofPos: -1, withRoof: Vs_or, hookPos: -1, withHook: Vs_oh),
    VowelSeqInfo(length: 1, complete: 1, conSuffix: 1, v: [Vnl_or, VnlNonVnChar, VnlNonVnChar], sub: [Vs_or, VsNil, VsNil], roofPos: 0, withRoof: VsNil, hookPos: -1, withHook: Vs_oh),
    VowelSeqInfo(length: 1, complete: 1, conSuffix: 1, v: [Vnl_oh, VnlNonVnChar, VnlNonVnChar], sub: [Vs_oh, VsNil, VsNil], roofPos: -1, withRoof: Vs_or, hookPos: 0, withHook: VsNil),
    VowelSeqInfo(length: 1, complete: 1, conSuffix: 1, v: [Vnl_u, VnlNonVnChar, VnlNonVnChar], sub: [Vs_u, VsNil, VsNil], roofPos: -1, withRoof: VsNil, hookPos: -1, withHook: Vs_uh),
    VowelSeqInfo(length: 1, complete: 1, conSuffix: 1, v: [Vnl_uh, VnlNonVnChar, VnlNonVnChar], sub: [Vs_uh, VsNil, VsNil], roofPos: -1, withRoof: VsNil, hookPos: 0, withHook: VsNil),
    VowelSeqInfo(length: 1, complete: 1, conSuffix: 1, v: [Vnl_y, VnlNonVnChar, VnlNonVnChar], sub: [Vs_y, VsNil, VsNil], roofPos: -1, withRoof: VsNil, hookPos: -1, withHook: VsNil),
    VowelSeqInfo(length: 2, complete: 1, conSuffix: 0, v: [Vnl_a, Vnl_i, VnlNonVnChar], sub: [Vs_a, Vs_ai, VsNil], roofPos: -1, withRoof: VsNil, hookPos: -1, withHook: VsNil),
    VowelSeqInfo(length: 2, complete: 1, conSuffix: 0, v: [Vnl_a, Vnl_o, VnlNonVnChar], sub: [Vs_a, Vs_ao, VsNil], roofPos: -1, withRoof: VsNil, hookPos: -1, withHook: VsNil),
    VowelSeqInfo(length: 2, complete: 1, conSuffix: 0, v: [Vnl_a, Vnl_u, VnlNonVnChar], sub: [Vs_a, Vs_au, VsNil], roofPos: -1, withRoof: Vs_aru, hookPos: -1, withHook: VsNil),
    VowelSeqInfo(length: 2, complete: 1, conSuffix: 0, v: [Vnl_a, Vnl_y, VnlNonVnChar], sub: [Vs_a, Vs_ay, VsNil], roofPos: -1, withRoof: Vs_ary, hookPos: -1, withHook: VsNil),
    VowelSeqInfo(length: 2, complete: 1, conSuffix: 0, v: [Vnl_ar, Vnl_u, VnlNonVnChar], sub: [Vs_ar, Vs_aru, VsNil], roofPos: 0, withRoof: VsNil, hookPos: -1, withHook: VsNil),
    VowelSeqInfo(length: 2, complete: 1, conSuffix: 0, v: [Vnl_ar, Vnl_y, VnlNonVnChar], sub: [Vs_ar, Vs_ary, VsNil], roofPos: 0, withRoof: VsNil, hookPos: -1, withHook: VsNil),
    VowelSeqInfo(length: 2, complete: 1, conSuffix: 0, v: [Vnl_e, Vnl_o, VnlNonVnChar], sub: [Vs_e, Vs_eo, VsNil], roofPos: -1, withRoof: VsNil, hookPos: -1, withHook: VsNil),
    VowelSeqInfo(length: 2, complete: 0, conSuffix: 0, v: [Vnl_e, Vnl_u, VnlNonVnChar], sub: [Vs_e, Vs_eu, VsNil], roofPos: -1, withRoof: Vs_eru, hookPos: -1, withHook: VsNil),
    VowelSeqInfo(length: 2, complete: 1, conSuffix: 0, v: [Vnl_er, Vnl_u, VnlNonVnChar], sub: [Vs_er, Vs_eru, VsNil], roofPos: 0, withRoof: VsNil, hookPos: -1, withHook: VsNil),
    VowelSeqInfo(length: 2, complete: 1, conSuffix: 1, v: [Vnl_i, Vnl_a, VnlNonVnChar], sub: [Vs_i, Vs_ia, VsNil], roofPos: -1, withRoof: VsNil, hookPos: -1, withHook: VsNil),
    VowelSeqInfo(length: 2, complete: 0, conSuffix: 1, v: [Vnl_i, Vnl_e, VnlNonVnChar], sub: [Vs_i, Vs_ie, VsNil], roofPos: -1, withRoof: Vs_ier, hookPos: -1, withHook: VsNil),
    VowelSeqInfo(length: 2, complete: 1, conSuffix: 1, v: [Vnl_i, Vnl_er, VnlNonVnChar], sub: [Vs_i, Vs_ier, VsNil], roofPos: 1, withRoof: VsNil, hookPos: -1, withHook: VsNil),
    VowelSeqInfo(length: 2, complete: 1, conSuffix: 0, v: [Vnl_i, Vnl_u, VnlNonVnChar], sub: [Vs_i, Vs_iu, VsNil], roofPos: -1, withRoof: VsNil, hookPos: -1, withHook: VsNil),
    VowelSeqInfo(length: 2, complete: 1, conSuffix: 1, v: [Vnl_o, Vnl_a, VnlNonVnChar], sub: [Vs_o, Vs_oa, VsNil], roofPos: -1, withRoof: VsNil, hookPos: -1, withHook: Vs_oab),
    VowelSeqInfo(length: 2, complete: 1, conSuffix: 1, v: [Vnl_o, Vnl_ab, VnlNonVnChar], sub: [Vs_o, Vs_oab, VsNil], roofPos: -1, withRoof: VsNil, hookPos: 1, withHook: VsNil),
    VowelSeqInfo(length: 2, complete: 1, conSuffix: 1, v: [Vnl_o, Vnl_e, VnlNonVnChar], sub: [Vs_o, Vs_oe, VsNil], roofPos: -1, withRoof: VsNil, hookPos: -1, withHook: VsNil),
    VowelSeqInfo(length: 2, complete: 1, conSuffix: 0, v: [Vnl_o, Vnl_i, VnlNonVnChar], sub: [Vs_o, Vs_oi, VsNil], roofPos: -1, withRoof: Vs_ori, hookPos: -1, withHook: Vs_ohi),
    VowelSeqInfo(length: 2, complete: 1, conSuffix: 0, v: [Vnl_or, Vnl_i, VnlNonVnChar], sub: [Vs_or, Vs_ori, VsNil], roofPos: 0, withRoof: VsNil, hookPos: -1, withHook: Vs_ohi),
    VowelSeqInfo(length: 2, complete: 1, conSuffix: 0, v: [Vnl_oh, Vnl_i, VnlNonVnChar], sub: [Vs_oh, Vs_ohi, VsNil], roofPos: -1, withRoof: Vs_ori, hookPos: 0, withHook: VsNil),
    VowelSeqInfo(length: 2, complete: 1, conSuffix: 1, v: [Vnl_u, Vnl_a, VnlNonVnChar], sub: [Vs_u, Vs_ua, VsNil], roofPos: -1, withRoof: Vs_uar, hookPos: -1, withHook: Vs_uha),
    VowelSeqInfo(length: 2, complete: 1, conSuffix: 1, v: [Vnl_u, Vnl_ar, VnlNonVnChar], sub: [Vs_u, Vs_uar, VsNil], roofPos: 1, withRoof: VsNil, hookPos: -1, withHook: VsNil),
    VowelSeqInfo(length: 2, complete: 0, conSuffix: 1, v: [Vnl_u, Vnl_e, VnlNonVnChar], sub: [Vs_u, Vs_ue, VsNil], roofPos: -1, withRoof: Vs_uer, hookPos: -1, withHook: VsNil),
    VowelSeqInfo(length: 2, complete: 1, conSuffix: 1, v: [Vnl_u, Vnl_er, VnlNonVnChar], sub: [Vs_u, Vs_uer, VsNil], roofPos: 1, withRoof: VsNil, hookPos: -1, withHook: VsNil),
    VowelSeqInfo(length: 2, complete: 1, conSuffix: 0, v: [Vnl_u, Vnl_i, VnlNonVnChar], sub: [Vs_u, Vs_ui, VsNil], roofPos: -1, withRoof: VsNil, hookPos: -1, withHook: Vs_uhi),
    VowelSeqInfo(length: 2, complete: 0, conSuffix: 1, v: [Vnl_u, Vnl_o, VnlNonVnChar], sub: [Vs_u, Vs_uo, VsNil], roofPos: -1, withRoof: Vs_uor, hookPos: -1, withHook: Vs_uho),
    VowelSeqInfo(length: 2, complete: 1, conSuffix: 1, v: [Vnl_u, Vnl_or, VnlNonVnChar], sub: [Vs_u, Vs_uor, VsNil], roofPos: 1, withRoof: VsNil, hookPos: -1, withHook: Vs_uoh),
    VowelSeqInfo(length: 2, complete: 1, conSuffix: 1, v: [Vnl_u, Vnl_oh, VnlNonVnChar], sub: [Vs_u, Vs_uoh, VsNil], roofPos: -1, withRoof: Vs_uor, hookPos: 1, withHook: Vs_uhoh),
    VowelSeqInfo(length: 2, complete: 0, conSuffix: 0, v: [Vnl_u, Vnl_u, VnlNonVnChar], sub: [Vs_u, Vs_uu, VsNil], roofPos: -1, withRoof: VsNil, hookPos: -1, withHook: Vs_uhu),
    VowelSeqInfo(length: 2, complete: 1, conSuffix: 1, v: [Vnl_u, Vnl_y, VnlNonVnChar], sub: [Vs_u, Vs_uy, VsNil], roofPos: -1, withRoof: VsNil, hookPos: -1, withHook: VsNil),
    VowelSeqInfo(length: 2, complete: 1, conSuffix: 0, v: [Vnl_uh, Vnl_a, VnlNonVnChar], sub: [Vs_uh, Vs_uha, VsNil], roofPos: -1, withRoof: VsNil, hookPos: 0, withHook: VsNil),
    VowelSeqInfo(length: 2, complete: 1, conSuffix: 0, v: [Vnl_uh, Vnl_i, VnlNonVnChar], sub: [Vs_uh, Vs_uhi, VsNil], roofPos: -1, withRoof: VsNil, hookPos: 0, withHook: VsNil),
    VowelSeqInfo(length: 2, complete: 0, conSuffix: 1, v: [Vnl_uh, Vnl_o, VnlNonVnChar], sub: [Vs_uh, Vs_uho, VsNil], roofPos: -1, withRoof: VsNil, hookPos: 0, withHook: Vs_uhoh),
    VowelSeqInfo(length: 2, complete: 1, conSuffix: 1, v: [Vnl_uh, Vnl_oh, VnlNonVnChar], sub: [Vs_uh, Vs_uhoh, VsNil], roofPos: -1, withRoof: VsNil, hookPos: 0, withHook: VsNil),
    VowelSeqInfo(length: 2, complete: 1, conSuffix: 0, v: [Vnl_uh, Vnl_u, VnlNonVnChar], sub: [Vs_uh, Vs_uhu, VsNil], roofPos: -1, withRoof: VsNil, hookPos: 0, withHook: VsNil),
    VowelSeqInfo(length: 2, complete: 0, conSuffix: 1, v: [Vnl_y, Vnl_e, VnlNonVnChar], sub: [Vs_y, Vs_ye, VsNil], roofPos: -1, withRoof: Vs_yer, hookPos: -1, withHook: VsNil),
    VowelSeqInfo(length: 2, complete: 1, conSuffix: 1, v: [Vnl_y, Vnl_er, VnlNonVnChar], sub: [Vs_y, Vs_yer, VsNil], roofPos: 1, withRoof: VsNil, hookPos: -1, withHook: VsNil),
    VowelSeqInfo(length: 3, complete: 0, conSuffix: 0, v: [Vnl_i, Vnl_e, Vnl_u], sub: [Vs_i, Vs_ie, Vs_ieu], roofPos: -1, withRoof: Vs_ieru, hookPos: -1, withHook: VsNil),
    VowelSeqInfo(length: 3, complete: 1, conSuffix: 0, v: [Vnl_i, Vnl_er, Vnl_u], sub: [Vs_i, Vs_ier, Vs_ieru], roofPos: 1, withRoof: VsNil, hookPos: -1, withHook: VsNil),
    VowelSeqInfo(length: 3, complete: 1, conSuffix: 0, v: [Vnl_o, Vnl_a, Vnl_i], sub: [Vs_o, Vs_oa, Vs_oai], roofPos: -1, withRoof: VsNil, hookPos: -1, withHook: VsNil),
    VowelSeqInfo(length: 3, complete: 1, conSuffix: 0, v: [Vnl_o, Vnl_a, Vnl_y], sub: [Vs_o, Vs_oa, Vs_oay], roofPos: -1, withRoof: VsNil, hookPos: -1, withHook: VsNil),
    VowelSeqInfo(length: 3, complete: 1, conSuffix: 0, v: [Vnl_o, Vnl_e, Vnl_o], sub: [Vs_o, Vs_oe, Vs_oeo], roofPos: -1, withRoof: VsNil, hookPos: -1, withHook: VsNil),
    VowelSeqInfo(length: 3, complete: 0, conSuffix: 0, v: [Vnl_u, Vnl_a, Vnl_y], sub: [Vs_u, Vs_ua, Vs_uay], roofPos: -1, withRoof: Vs_uary, hookPos: -1, withHook: VsNil),
    VowelSeqInfo(length: 3, complete: 1, conSuffix: 0, v: [Vnl_u, Vnl_ar, Vnl_y], sub: [Vs_u, Vs_uar, Vs_uary], roofPos: 1, withRoof: VsNil, hookPos: -1, withHook: VsNil),
    VowelSeqInfo(length: 3, complete: 0, conSuffix: 0, v: [Vnl_u, Vnl_o, Vnl_i], sub: [Vs_u, Vs_uo, Vs_uoi], roofPos: -1, withRoof: Vs_uori, hookPos: -1, withHook: Vs_uhoi),
    VowelSeqInfo(length: 3, complete: 0, conSuffix: 0, v: [Vnl_u, Vnl_o, Vnl_u], sub: [Vs_u, Vs_uo, Vs_uou], roofPos: -1, withRoof: VsNil, hookPos: -1, withHook: Vs_uhou),
    VowelSeqInfo(length: 3, complete: 1, conSuffix: 0, v: [Vnl_u, Vnl_or, Vnl_i], sub: [Vs_u, Vs_uor, Vs_uori], roofPos: 1, withRoof: VsNil, hookPos: -1, withHook: Vs_uohi),
    VowelSeqInfo(length: 3, complete: 0, conSuffix: 0, v: [Vnl_u, Vnl_oh, Vnl_i], sub: [Vs_u, Vs_uoh, Vs_uohi], roofPos: -1, withRoof: Vs_uori, hookPos: 1, withHook: Vs_uhohi),
    VowelSeqInfo(length: 3, complete: 0, conSuffix: 0, v: [Vnl_u, Vnl_oh, Vnl_u], sub: [Vs_u, Vs_uoh, Vs_uohu], roofPos: -1, withRoof: VsNil, hookPos: 1, withHook: Vs_uhohu),
    VowelSeqInfo(length: 3, complete: 1, conSuffix: 0, v: [Vnl_u, Vnl_y, Vnl_a], sub: [Vs_u, Vs_uy, Vs_uya], roofPos: -1, withRoof: VsNil, hookPos: -1, withHook: VsNil),
    VowelSeqInfo(length: 3, complete: 0, conSuffix: 1, v: [Vnl_u, Vnl_y, Vnl_e], sub: [Vs_u, Vs_uy, Vs_uye], roofPos: -1, withRoof: Vs_uyer, hookPos: -1, withHook: VsNil),
    VowelSeqInfo(length: 3, complete: 1, conSuffix: 1, v: [Vnl_u, Vnl_y, Vnl_er], sub: [Vs_u, Vs_uy, Vs_uyer], roofPos: 2, withRoof: VsNil, hookPos: -1, withHook: VsNil),
    VowelSeqInfo(length: 3, complete: 1, conSuffix: 0, v: [Vnl_u, Vnl_y, Vnl_u], sub: [Vs_u, Vs_uy, Vs_uyu], roofPos: -1, withRoof: VsNil, hookPos: -1, withHook: VsNil),
    VowelSeqInfo(length: 3, complete: 0, conSuffix: 0, v: [Vnl_uh, Vnl_o, Vnl_i], sub: [Vs_uh, Vs_uho, Vs_uhoi], roofPos: -1, withRoof: VsNil, hookPos: 0, withHook: Vs_uhohi),
    VowelSeqInfo(length: 3, complete: 0, conSuffix: 0, v: [Vnl_uh, Vnl_o, Vnl_u], sub: [Vs_uh, Vs_uho, Vs_uhou], roofPos: -1, withRoof: VsNil, hookPos: 0, withHook: Vs_uhohu),
    VowelSeqInfo(length: 3, complete: 1, conSuffix: 0, v: [Vnl_uh, Vnl_oh, Vnl_i], sub: [Vs_uh, Vs_uhoh, Vs_uhohi], roofPos: -1, withRoof: VsNil, hookPos: 0, withHook: VsNil),
    VowelSeqInfo(length: 3, complete: 1, conSuffix: 0, v: [Vnl_uh, Vnl_oh, Vnl_u], sub: [Vs_uh, Vs_uhoh, Vs_uhohu], roofPos: -1, withRoof: VsNil, hookPos: 0, withHook: VsNil),
    VowelSeqInfo(length: 3, complete: 0, conSuffix: 0, v: [Vnl_y, Vnl_e, Vnl_u], sub: [Vs_y, Vs_ye, Vs_yeu], roofPos: -1, withRoof: Vs_yeru, hookPos: -1, withHook: VsNil),
    VowelSeqInfo(length: 3, complete: 1, conSuffix: 0, v: [Vnl_y, Vnl_er, Vnl_u], sub: [Vs_y, Vs_yer, Vs_yeru], roofPos: 1, withRoof: VsNil, hookPos: -1, withHook: VsNil),
]

struct ConSeqInfo {
    let length: Int
    let c: [VnLexiName] // [3]
    let suffix: Bool
}

let cSeqList: [ConSeqInfo] = [
    ConSeqInfo(length: 1, c: [Vnl_b, VnlNonVnChar, VnlNonVnChar], suffix: false),
    ConSeqInfo(length: 1, c: [Vnl_c, VnlNonVnChar, VnlNonVnChar], suffix: true),
    ConSeqInfo(length: 2, c: [Vnl_c, Vnl_h, VnlNonVnChar], suffix: true),
    ConSeqInfo(length: 1, c: [Vnl_d, VnlNonVnChar, VnlNonVnChar], suffix: false),
    ConSeqInfo(length: 1, c: [Vnl_dd, VnlNonVnChar, VnlNonVnChar], suffix: false),
    ConSeqInfo(length: 2, c: [Vnl_d, Vnl_z, VnlNonVnChar], suffix: false),
    ConSeqInfo(length: 1, c: [Vnl_g, VnlNonVnChar, VnlNonVnChar], suffix: false),
    ConSeqInfo(length: 2, c: [Vnl_g, Vnl_h, VnlNonVnChar], suffix: false),
    ConSeqInfo(length: 2, c: [Vnl_g, Vnl_i, VnlNonVnChar], suffix: false),
    ConSeqInfo(length: 3, c: [Vnl_g, Vnl_i, Vnl_n], suffix: false),
    ConSeqInfo(length: 1, c: [Vnl_k, VnlNonVnChar, VnlNonVnChar], suffix: false),
    ConSeqInfo(length: 2, c: [Vnl_k, Vnl_h, VnlNonVnChar], suffix: false),
    ConSeqInfo(length: 1, c: [Vnl_l, VnlNonVnChar, VnlNonVnChar], suffix: false),
    ConSeqInfo(length: 1, c: [Vnl_m, VnlNonVnChar, VnlNonVnChar], suffix: true),
    ConSeqInfo(length: 1, c: [Vnl_n, VnlNonVnChar, VnlNonVnChar], suffix: true),
    ConSeqInfo(length: 2, c: [Vnl_n, Vnl_g, VnlNonVnChar], suffix: true),
    ConSeqInfo(length: 3, c: [Vnl_n, Vnl_g, Vnl_h], suffix: false),
    ConSeqInfo(length: 2, c: [Vnl_n, Vnl_h, VnlNonVnChar], suffix: true),
    ConSeqInfo(length: 1, c: [Vnl_p, VnlNonVnChar, VnlNonVnChar], suffix: true),
    ConSeqInfo(length: 2, c: [Vnl_p, Vnl_h, VnlNonVnChar], suffix: false),
    ConSeqInfo(length: 1, c: [Vnl_q, VnlNonVnChar, VnlNonVnChar], suffix: false),
    ConSeqInfo(length: 2, c: [Vnl_q, Vnl_u, VnlNonVnChar], suffix: false),
    ConSeqInfo(length: 1, c: [Vnl_r, VnlNonVnChar, VnlNonVnChar], suffix: false),
    ConSeqInfo(length: 1, c: [Vnl_s, VnlNonVnChar, VnlNonVnChar], suffix: false),
    ConSeqInfo(length: 1, c: [Vnl_t, VnlNonVnChar, VnlNonVnChar], suffix: true),
    ConSeqInfo(length: 2, c: [Vnl_t, Vnl_h, VnlNonVnChar], suffix: false),
    ConSeqInfo(length: 2, c: [Vnl_t, Vnl_r, VnlNonVnChar], suffix: false),
    ConSeqInfo(length: 1, c: [Vnl_v, VnlNonVnChar, VnlNonVnChar], suffix: false),
    ConSeqInfo(length: 1, c: [Vnl_x, VnlNonVnChar, VnlNonVnChar], suffix: false),
]

// (vowel seq, consonant seq) pairs; packed into Int set: v*64+c
private let vcPairList: [Int] = [
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
]

let vcPairSet: Set<Int> = Set(vcPairList)

// lookup structures — packed key: (a+1)*65536 + (b+1)*256 + (c+1)  (VnlNonVnChar=-1 -> 0)
private func pack3(_ a: Int, _ b: Int, _ c: Int) -> Int { (a + 1) * 65536 + (b + 1) * 256 + (c + 1) }

let vSeqLookup: [Int: VowelSeq] = {
    var m = [Int: VowelSeq](minimumCapacity: vSeqList.count)
    for (i, info) in vSeqList.enumerated() {
        m[pack3(info.v[0], info.v[1], info.v[2])] = i
    }
    return m
}()

let cSeqLookup: [Int: ConSeq] = {
    var m = [Int: ConSeq](minimumCapacity: cSeqList.count)
    for (i, info) in cSeqList.enumerated() {
        m[pack3(info.c[0], info.c[1], info.c[2])] = i
    }
    return m
}()

func lookupVSeq(_ v1: VnLexiName, _ v2: VnLexiName, _ v3: VnLexiName) -> VowelSeq {
    vSeqLookup[pack3(v1, v2, v3)] ?? VsNil
}

func lookupCSeq(_ c1: VnLexiName, _ c2: VnLexiName, _ c3: VnLexiName) -> ConSeq {
    cSeqLookup[pack3(c1, c2, c3)] ?? CsNil
}

func isVnVowel(_ sym: VnLexiName) -> Bool {
    sym >= 0 && sym < isVnVowelTab.count && isVnVowelTab[sym]
}

// A-Z -> VnLexiName (inputproc.cpp AZLexiUpper/AZLexiLower)
let azLexiUpper: [Int] = [
    VnlA, VnlB, VnlC, VnlD, VnlE, VnlF, VnlG, VnlH, VnlI, VnlJ,
    VnlK, VnlL, VnlM, VnlN, VnlO, VnlP, VnlQ, VnlR, VnlS, VnlT,
    VnlU, VnlV, VnlW, VnlX, VnlY, VnlZ,
]
let azLexiLower: [Int] = [
    Vnl_a, Vnl_b, Vnl_c, Vnl_d, Vnl_e, Vnl_f, Vnl_g, Vnl_h, Vnl_i, Vnl_j,
    Vnl_k, Vnl_l, Vnl_m, Vnl_n, Vnl_o, Vnl_p, Vnl_q, Vnl_r, Vnl_s, Vnl_t,
    Vnl_u, Vnl_v, Vnl_w, Vnl_x, Vnl_y, Vnl_z,
]

let isVnVowelTab: [Bool] = {
    var t = [Bool](repeating: true, count: 186)
    for ch in "a".ord..."z".ord {
        if ch != "a".ord && ch != "e".ord && ch != "i".ord &&
           ch != "o".ord && ch != "u".ord && ch != "y".ord {
            t[azLexiLower[ch - "a".ord]] = false
            t[azLexiUpper[ch - "a".ord]] = false
        }
    }
    t[Vnl_dd] = false
    t[VnlDD] = false
    return t
}()

func changeCase(_ x: VnLexiName) -> VnLexiName {
    if x == VnlNonVnChar { return x }
    return (x & 0x01) == 0 ? x + 1 : x - 1
}

func vnToLower(_ x: VnLexiName) -> VnLexiName {
    if x == VnlNonVnChar { return x }
    return (x & 0x01) == 0 ? x + 1 : x // even -> lower
}

// ---------- key maps (inputproc.cpp) ----------

struct KeyMapping {
    let key: Int
    let action: Int
}

let telexMethodMapping: [KeyMapping] = [
    KeyMapping(key: "Z".ord, action: VneTone0),
    KeyMapping(key: "S".ord, action: VneTone1),
    KeyMapping(key: "F".ord, action: VneTone2),
    KeyMapping(key: "R".ord, action: VneTone3),
    KeyMapping(key: "X".ord, action: VneTone4),
    KeyMapping(key: "J".ord, action: VneTone5),
    KeyMapping(key: "W".ord, action: VneTelexW),
    KeyMapping(key: "A".ord, action: VneRoofA),
    KeyMapping(key: "E".ord, action: VneRoofE),
    KeyMapping(key: "O".ord, action: VneRoofO),
    KeyMapping(key: "D".ord, action: VneDd),
    KeyMapping(key: "[".ord, action: VneCount + Vnl_oh),
    KeyMapping(key: "]".ord, action: VneCount + Vnl_uh),
    KeyMapping(key: "{".ord, action: VneCount + VnlOh),
    KeyMapping(key: "}".ord, action: VneCount + VnlUh),
]

let simpleTelexMethodMapping: [KeyMapping] = [
    KeyMapping(key: "Z".ord, action: VneTone0),
    KeyMapping(key: "S".ord, action: VneTone1),
    KeyMapping(key: "F".ord, action: VneTone2),
    KeyMapping(key: "R".ord, action: VneTone3),
    KeyMapping(key: "X".ord, action: VneTone4),
    KeyMapping(key: "J".ord, action: VneTone5),
    KeyMapping(key: "W".ord, action: VneHookAll),
    KeyMapping(key: "A".ord, action: VneRoofA),
    KeyMapping(key: "E".ord, action: VneRoofE),
    KeyMapping(key: "O".ord, action: VneRoofO),
    KeyMapping(key: "D".ord, action: VneDd),
]

let vniMethodMapping: [KeyMapping] = [
    KeyMapping(key: "0".ord, action: VneTone0),
    KeyMapping(key: "1".ord, action: VneTone1),
    KeyMapping(key: "2".ord, action: VneTone2),
    KeyMapping(key: "3".ord, action: VneTone3),
    KeyMapping(key: "4".ord, action: VneTone4),
    KeyMapping(key: "5".ord, action: VneTone5),
    KeyMapping(key: "6".ord, action: VneRoofAll),
    KeyMapping(key: "7".ord, action: VneHookUO),
    KeyMapping(key: "8".ord, action: VneBowl),
    KeyMapping(key: "9".ord, action: VneDd),
]

let viqrMethodMapping: [KeyMapping] = [
    KeyMapping(key: "0".ord, action: VneTone0),
    KeyMapping(key: "'".ord, action: VneTone1),
    KeyMapping(key: "`".ord, action: VneTone2),
    KeyMapping(key: "?".ord, action: VneTone3),
    KeyMapping(key: "~".ord, action: VneTone4),
    KeyMapping(key: ".".ord, action: VneTone5),
    KeyMapping(key: "^".ord, action: VneRoofAll),
    KeyMapping(key: "+".ord, action: VneHookUO),
    KeyMapping(key: "*".ord, action: VneHookUO),
    KeyMapping(key: "(".ord, action: VneBowl),
    KeyMapping(key: "D".ord, action: VneDd),
    KeyMapping(key: "\\".ord, action: VneEscChar),
]

let msViMethodMapping: [KeyMapping] = [
    KeyMapping(key: "5".ord, action: VneTone2),
    KeyMapping(key: "%".ord, action: VneTone2),
    KeyMapping(key: "6".ord, action: VneTone3),
    KeyMapping(key: "^".ord, action: VneTone3),
    KeyMapping(key: "7".ord, action: VneTone4),
    KeyMapping(key: "&".ord, action: VneTone4),
    KeyMapping(key: "8".ord, action: VneTone1),
    KeyMapping(key: "*".ord, action: VneTone1),
    KeyMapping(key: "9".ord, action: VneTone5),
    KeyMapping(key: "(".ord, action: VneTone5),
    KeyMapping(key: "1".ord, action: VneCount + Vnl_ab),
    KeyMapping(key: "!".ord, action: VneCount + VnlAb),
    KeyMapping(key: "2".ord, action: VneCount + Vnl_ar),
    KeyMapping(key: "@".ord, action: VneCount + VnlAr),
    KeyMapping(key: "3".ord, action: VneCount + Vnl_er),
    KeyMapping(key: "#".ord, action: VneCount + VnlEr),
    KeyMapping(key: "4".ord, action: VneCount + Vnl_or),
    KeyMapping(key: "$".ord, action: VneCount + VnlOr),
    KeyMapping(key: "0".ord, action: VneCount + Vnl_dd),
    KeyMapping(key: ")".ord, action: VneCount + VnlDD),
    KeyMapping(key: "[".ord, action: VneCount + Vnl_uh),
    KeyMapping(key: "]".ord, action: VneCount + Vnl_oh),
    KeyMapping(key: "{".ord, action: VneCount + VnlUh),
    KeyMapping(key: "}".ord, action: VneCount + VnlOh),
]

let wordBreakSyms: [Int] = [
    ",".ord, ";".ord, ":".ord, ".".ord, "\"".ord, "'".ord, "!".ord, "?".ord, " ".ord,
    "<".ord, ">".ord, "=".ord, "+".ord, "-".ord, "*".ord, "/".ord, "\\".ord,
    "_".ord, "@".ord, "#".ord, "$".ord, "%".ord, "&".ord, "(".ord, ")".ord, "{".ord, "}".ord, "[".ord, "]".ord, "|".ord,
]

// ISO western chars that are also Vietnamese chars (inputproc.cpp AscVnLexiList;
// note the original has a duplicate 0xC2 entry — the last one (A4) wins)
private let ascVnLexiEntries: [Int] = [
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
]

// chars that have different code points in Unicode vs Western charsets
let specialWesternChars: [Int] = [
    0x80, 0x82, 0x83, 0x84, 0x85, 0x86, 0x87, 0x88,
    0x89, 0x8A, 0x8B, 0x8C, 0x8E, 0x91, 0x92, 0x93,
    0x94, 0x95, 0x96, 0x97, 0x98, 0x99, 0x9A, 0x9B,
    0x9C, 0x9E, 0x9F,
]

// ---------- classification maps (Go init()) ----------

let ukcMap: [Int] = { // UkCharType
    var t = [Int](repeating: 0, count: 256)
    for c in 0...32 { t[c] = UkcReset }
    for c in 33..<256 { t[c] = UkcNonVn }
    for c in "a".ord..."z".ord { t[c] = UkcVn }
    for c in "A".ord..."Z".ord { t[c] = UkcVn }
    var i = 0
    while i < ascVnLexiEntries.count {
        let asc = ascVnLexiEntries[i]
        if asc != 0 { t[asc] = UkcVn }
        i += 2
    }
    t["j".ord] = UkcNonVn
    t["J".ord] = UkcNonVn
    t["f".ord] = UkcNonVn
    t["F".ord] = UkcNonVn
    t["w".ord] = UkcNonVn
    t["W".ord] = UkcNonVn
    for s in wordBreakSyms { t[s] = UkcWordBreak }
    return t
}()

let isoVnLexiMap: [VnLexiName] = {
    var t = [VnLexiName](repeating: VnlNonVnChar, count: 256)
    var i = 0
    while i < ascVnLexiEntries.count {
        let asc = ascVnLexiEntries[i]
        if asc != 0 { t[asc] = ascVnLexiEntries[i + 1] }
        i += 2
    }
    for c in "a".ord..."z".ord { t[c] = azLexiLower[c - "a".ord] }
    for c in "A".ord..."Z".ord { t[c] = azLexiUpper[c - "A".ord] }
    return t
}()

func isoToVnLexi(_ keyCode: Int) -> VnLexiName {
    if keyCode > 255 || keyCode < 0 { return VnlNonVnChar }
    return isoVnLexiMap[keyCode]
}

// SetupUnikeyEngine — must run after isoVnLexiMap
let isoStdVnCharMap: [StdVnChar] = {
    var t = [StdVnChar](repeating: 0, count: 256)
    for i in 0..<256 { t[i] = i }
    for (i, ch) in specialWesternChars.enumerated() {
        t[ch] = VnlLastChar + i + VnStdCharOffset
    }
    for i in 0..<256 {
        let lexi = isoToVnLexi(i)
        if lexi != VnlNonVnChar {
            t[i] = lexi + VnStdCharOffset
        }
    }
    return t
}()

func isoToStdVnChar(_ keyCode: Int) -> StdVnChar {
    if keyCode >= 0 && keyCode < 256 { return isoStdVnCharMap[keyCode] }
    return keyCode
}

// ---------- Unicode table (vnconv/data.cpp), indexed by StdVnChar - VnStdCharOffset ----------
let unicodeTable: [UInt16] = [
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
]

let stdVnRootChar: [Int] = [
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
]

let stdVnNoTone: [Int] = [
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
]

// reverse map unicode -> StdVnChar (for macro file parsing)
let uniToStdVn: [UInt16: StdVnChar] = {
    var m = [UInt16: StdVnChar](minimumCapacity: TotalVnChars)
    for (i, u) in unicodeTable.enumerated() {
        if m[u] == nil { m[u] = i + VnStdCharOffset }
    }
    return m
}()

