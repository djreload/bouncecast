package uk.co.knrg.bouncecast.core.storage

import android.content.Context
import androidx.datastore.preferences.core.edit
import androidx.datastore.preferences.core.stringPreferencesKey
import androidx.datastore.preferences.preferencesDataStore
import kotlinx.coroutines.flow.first

private val Context.dataStore by preferencesDataStore(name = "bouncecast_mobile")

class ConfigStore(private val context: Context) {
    private val configKey = stringPreferencesKey("mobile_config_json")

    suspend fun readConfigJson(): String? {
        return context.dataStore.data.first()[configKey]
    }

    suspend fun writeConfigJson(json: String) {
        context.dataStore.edit { preferences ->
            preferences[configKey] = json
        }
    }
}

