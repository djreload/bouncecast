package uk.co.knrg.bouncecast.feature.ads

import org.junit.Assert.assertEquals
import org.junit.Test
import uk.co.knrg.bouncecast.core.model.AdConfig

class AdPriorityResolverTest {
    @Test
    fun resolvesFallbackOrder() {
        val providers = AdPriorityResolver.providers(
            AdConfig(
                enabled = true,
                fallbackEnabled = true,
                googleEnabled = true,
                unityEnabled = true,
                priority = listOf("unity", "google"),
            ),
        )

        assertEquals(listOf("unity", "google"), providers)
    }

    @Test
    fun disablesFallbackWhenConfigured() {
        val providers = AdPriorityResolver.providers(
            AdConfig(
                enabled = true,
                fallbackEnabled = false,
                googleEnabled = true,
                unityEnabled = true,
                priority = listOf("unity", "google"),
            ),
        )

        assertEquals(listOf("unity"), providers)
    }
}

