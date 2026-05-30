package uk.co.knrg.bouncecast.core.storage

import android.content.Context
import androidx.datastore.preferences.core.edit
import androidx.datastore.preferences.core.stringPreferencesKey
import androidx.datastore.preferences.preferencesDataStore
import kotlinx.coroutines.flow.first
import uk.co.knrg.bouncecast.core.model.ViewerIdentity

private val Context.dataStore by preferencesDataStore(name = "bouncecast_mobile")

class ConfigStore(private val context: Context) {
    private val configKey = stringPreferencesKey("mobile_config_json")
    private val viewerUserIdKey = stringPreferencesKey("viewer_user_id")
    private val viewerAccessTokenKey = stringPreferencesKey("viewer_access_token")
    private val viewerDisplayNameKey = stringPreferencesKey("viewer_display_name")

    suspend fun readConfigJson(): String? {
        return context.dataStore.data.first()[configKey]
    }

    suspend fun writeConfigJson(json: String) {
        context.dataStore.edit { preferences ->
            preferences[configKey] = json
        }
    }

    suspend fun readViewerIdentity(): ViewerIdentity {
        val preferences = context.dataStore.data.first()
        return ViewerIdentity(
            userId = preferences[viewerUserIdKey].orEmpty(),
            accessToken = preferences[viewerAccessTokenKey].orEmpty(),
            displayName = preferences[viewerDisplayNameKey].orEmpty().ifBlank { "Android Viewer" },
        )
    }

    suspend fun writeViewerIdentity(identity: ViewerIdentity) {
        val normalized = identity.normalized()
        context.dataStore.edit { preferences ->
            preferences[viewerUserIdKey] = normalized.userId
            preferences[viewerAccessTokenKey] = normalized.accessToken
            preferences[viewerDisplayNameKey] = normalized.displayName
        }
    }

    suspend fun clearViewerIdentity() {
        context.dataStore.edit { preferences ->
            preferences.remove(viewerUserIdKey)
            preferences.remove(viewerAccessTokenKey)
            preferences.remove(viewerDisplayNameKey)
        }
    }
}
