package uk.co.knrg.bouncecast.feature.ads

import android.view.View
import androidx.compose.runtime.Composable

interface AdProvider {
    val name: String
    fun isConfigured(): Boolean

    @Composable
    fun Banner(onFailed: () -> Unit): View?
}

