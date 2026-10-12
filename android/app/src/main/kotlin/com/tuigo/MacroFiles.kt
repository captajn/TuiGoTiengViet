package com.tuigo

import android.content.Context
import android.util.AtomicFile
import java.io.File
import java.io.FileNotFoundException

internal object MacroFiles {
    fun read(context: Context): String {
        val file = File(context.filesDir, "macro.txt")
        try {
            // openRead also recovers a .bak left by an interrupted atomic write.
            return AtomicFile(file).openRead().bufferedReader(Charsets.UTF_8).use { it.readText() }
        } catch (error: FileNotFoundException) {
            if (!file.exists() && !File(file.path + ".bak").exists()) return ""
            throw error
        }
    }
}
