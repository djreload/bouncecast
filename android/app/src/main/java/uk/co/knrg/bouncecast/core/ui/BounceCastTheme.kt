package uk.co.knrg.bouncecast.core.ui

import android.graphics.Color
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color as ComposeColor
import uk.co.knrg.bouncecast.core.model.MobileConfig

@Composable
fun BounceCastTheme(
    config: MobileConfig?,
    content: @Composable () -> Unit,
) {
    val primary = config?.branding?.primaryColor?.toComposeColor() ?: ComposeColor(0xFF080711)
    val accent = config?.branding?.accentColor?.toComposeColor() ?: ComposeColor(0xFFFF2A8A)
    val background = config?.branding?.backgroundColor?.toComposeColor() ?: ComposeColor(0xFF05050F)
    val dark = config?.branding?.themeMode != "light"
    val colors = if (dark) {
        darkColorScheme(
            primary = accent,
            secondary = primary,
            background = background,
            surface = ComposeColor(0xFF111124),
        )
    } else {
        lightColorScheme(
            primary = accent,
            secondary = primary,
            background = ComposeColor.White,
            surface = ComposeColor(0xFFF7F7FB),
        )
    }

    MaterialTheme(
        colorScheme = colors,
        content = content,
    )
}

private fun String.toComposeColor(): ComposeColor? = runCatching {
    ComposeColor(Color.parseColor(this))
}.getOrNull()

