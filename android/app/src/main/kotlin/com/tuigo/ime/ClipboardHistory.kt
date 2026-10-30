package com.tuigo.ime

import android.content.ClipData
import android.content.ClipDescription
import android.content.Context
import android.graphics.Bitmap
import android.graphics.BitmapFactory
import com.tuigo.Prefs
import org.json.JSONArray
import org.json.JSONObject
import java.io.File
import java.util.concurrent.Executors

/**
 * Samsung-style on-device clipboard history. Android exposes only the latest
 * clip, so the IME records each primary-clip change itself while it runs.
 * Sensitive clips (EXTRA_IS_SENSITIVE) and copies made in password fields are
 * never stored. Images are copied into filesDir because URI grants expire.
 *
 * Storage: filesDir/clipboard_history.json (newest first) + PNG files under
 * filesDir/clips/.
 * Pinned entries are kept above the cap; at most clipboardHistoryMax
 * (Prefs.KEY_CLIPBOARD_MAX) unpinned entries.
 */
object ClipboardHistory {
    const val DEFAULT_MAX_ENTRIES = 50
    private const val MAX_PINNED = 20
    private const val FILE = "clipboard_history.json"
    private const val IMGDIR = "clips"
    private const val MAX_IMAGE_EDGE = 768
    private const val MAX_IMAGE_BYTES = 1_500_000

    class Entry(
        val id: Long,
        val text: String?,
        val image: String?,
        val pinned: Boolean,
        val ts: Long,
    )

    @Volatile private var cache: MutableList<Entry>? = null
    private val imageExecutor = Executors.newSingleThreadExecutor()

    @Synchronized
    fun entries(ctx: Context): List<Entry> {
        val list = cache ?: loadLocked(ctx)
        // The user may have lowered the cap since the last insert — trim now
        // so the panel never shows more than the configured limit.
        if (enforceCapsLocked(ctx, list)) saveLocked(ctx, list)
        return ArrayList(list)
    }

    @Synchronized
    fun hasEntries(ctx: Context): Boolean =
        (cache ?: loadLocked(ctx)).isNotEmpty()

    /** Record the current primary clip. Safe to call from the listener on the
     *  main thread — image decoding runs on a background executor. */
    fun record(ctx: Context, clip: ClipData?, desc: ClipDescription?) {
        if (clip == null || clip.itemCount <= 0) return
        if (desc?.extras?.getBoolean("android.content.extra.IS_SENSITIVE") == true) return
        var text: String? = null
        var imageUri: android.net.Uri? = null
        for (i in 0 until clip.itemCount) {
            val item = clip.getItemAt(i) ?: continue
            if (text == null) {
                val t = item.text?.toString()
                if (!t.isNullOrBlank()) text = t
            }
            if (imageUri == null && item.uri != null) {
                val mime = (0 until (desc?.mimeTypeCount ?: 0))
                    .mapNotNull { desc?.getMimeType(it) }
                    .firstOrNull { it.startsWith("image/") }
                    ?: try { ctx.contentResolver.getType(item.uri) } catch (_: Exception) { null }
                if (mime != null && mime.startsWith("image/")) imageUri = item.uri
            }
        }
        if (text == null && imageUri == null) return
        val id = System.currentTimeMillis()
        val uri = imageUri
        if (uri == null) {
            insert(ctx, Entry(id, text, null, false, id))
        } else {
            // Decode/copy off the main thread, then record the file name.
            imageExecutor.execute {
                val name = saveImage(ctx, uri, id) ?: return@execute
                insert(ctx, Entry(id, null, name, false, id))
            }
        }
    }

    @Synchronized
    fun setPinned(ctx: Context, id: Long, pinned: Boolean) {
        val list = cache ?: loadLocked(ctx)
        val idx = list.indexOfFirst { it.id == id }
        if (idx < 0) return
        val e = list[idx]
        list[idx] = Entry(e.id, e.text, e.image, pinned, e.ts)
        saveLocked(ctx, list)
    }

    @Synchronized
    fun remove(ctx: Context, id: Long) {
        val list = cache ?: loadLocked(ctx)
        val idx = list.indexOfFirst { it.id == id }
        if (idx < 0) return
        list[idx].image?.let { File(imageDir(ctx), it).delete() }
        list.removeAt(idx)
        saveLocked(ctx, list)
    }

    @Synchronized
    fun clear(ctx: Context) {
        imageDir(ctx).listFiles()?.forEach { it.delete() }
        cache = mutableListOf()
        saveLocked(ctx, cache!!)
    }

    /** Drop the in-memory copy; next access reloads from disk. Called when the
     *  keyboard hides so the IME doesn't hold the whole history while idle. */
    @Synchronized
    fun unload() {
        cache = null
    }

    private fun imageDir(ctx: Context): File =
        File(ctx.filesDir, IMGDIR).apply { mkdirs() }

    fun imageFile(ctx: Context, name: String): File = File(imageDir(ctx), name)

    // ---- internals ------------------------------------------------------------

    private fun loadLocked(ctx: Context): MutableList<Entry> {
        val list = mutableListOf<Entry>()
        try {
            val arr = JSONArray(File(ctx.filesDir, FILE).readText())
            for (i in 0 until arr.length()) {
                val o = arr.optJSONObject(i) ?: continue
                val id = o.optLong("id", 0L)
                if (id <= 0) continue
                list += Entry(
                    id = id,
                    text = if (o.isNull("text")) null else o.optString("text"),
                    image = if (o.isNull("image")) null else o.optString("image"),
                    pinned = o.optBoolean("pinned", false),
                    ts = o.optLong("ts", id),
                )
            }
        } catch (_: Exception) { }
        // Newest first; pinned entries float to the top of the panel.
        list.sortWith(compareByDescending<Entry> { it.pinned }.thenByDescending { it.ts })
        cache = list
        return list
    }

    /** Insert newest-first; a re-copied text bumps to the top instead of
     *  duplicating. Oldest unpinned entries are evicted beyond the cap. */
    @Synchronized
    private fun insert(ctx: Context, e: Entry) {
        val list = cache ?: loadLocked(ctx)
        if (e.text != null) list.removeAll { it.text == e.text }
        list.add(e)
        list.sortWith(compareByDescending<Entry> { it.pinned }.thenByDescending { it.ts })
        enforceCapsLocked(ctx, list)
        saveLocked(ctx, list)
    }

    /** Evict oldest unpinned beyond the configured cap, then oldest pinned
     *  beyond cap + MAX_PINNED. Returns true when entries were removed. */
    private fun enforceCapsLocked(ctx: Context, list: MutableList<Entry>): Boolean {
        val cap = Prefs.clipboardHistoryMax(ctx)
        var changed = false
        fun evictAt(idx: Int) {
            if (idx < 0) return
            list[idx].image?.let { File(imageDir(ctx), it).delete() }
            list.removeAt(idx)
            changed = true
        }
        while (list.count { !it.pinned } > cap) evictAt(list.indexOfLast { !it.pinned })
        while (list.size > cap + MAX_PINNED) evictAt(list.indexOfLast { it.pinned })
        return changed
    }

    private fun saveLocked(ctx: Context, list: List<Entry>) {
        val arr = JSONArray()
        for (e in list) {
            arr.put(JSONObject()
                .put("id", e.id)
                .put("text", e.text)
                .put("image", e.image)
                .put("pinned", e.pinned)
                .put("ts", e.ts))
        }
        // Atomic replace: write temp then rename so a crash mid-write keeps the
        // previous file intact.
        val f = File(ctx.filesDir, FILE)
        val tmp = File(ctx.filesDir, "$FILE.tmp")
        try {
            tmp.writeText(arr.toString())
            if (!tmp.renameTo(f)) {
                f.delete()
                tmp.renameTo(f)
            }
        } catch (_: Exception) { }
    }

    private fun saveImage(ctx: Context, uri: android.net.Uri, id: Long): String? = try {
        val bounds = BitmapFactory.Options().apply { inJustDecodeBounds = true }
        ctx.contentResolver.openInputStream(uri)?.use {
            BitmapFactory.decodeStream(it, null, bounds)
        }
        if (bounds.outWidth <= 0 || bounds.outHeight <= 0) null
        else {
            var sample = 1
            while (bounds.outWidth / sample > MAX_IMAGE_EDGE ||
                   bounds.outHeight / sample > MAX_IMAGE_EDGE) sample *= 2
            val opts = BitmapFactory.Options().apply { inSampleSize = sample }
            val bmp = ctx.contentResolver.openInputStream(uri)?.use {
                BitmapFactory.decodeStream(it, null, opts)
            } ?: return null
            val name = "c_$id.png"
            val out = File(imageDir(ctx), name)
            out.outputStream().use { bmp.compress(Bitmap.CompressFormat.PNG, 100, it) }
            if (out.length() > MAX_IMAGE_BYTES) { out.delete(); null } else name
        }
    } catch (_: Exception) { null }
}
