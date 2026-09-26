package com.sinirbekleme

import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.lifecycle.viewmodel.compose.viewModel
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.rememberNavController
import androidx.navigation.toRoute
import com.sinirbekleme.di.AppContainer
import com.sinirbekleme.domain.model.Direction
import com.sinirbekleme.presentation.detail.DetailScreen
import com.sinirbekleme.presentation.detail.DetailViewModel
import com.sinirbekleme.presentation.home.HomeScreen
import com.sinirbekleme.presentation.home.HomeViewModel
import com.sinirbekleme.presentation.theme.AppTheme
import kotlinx.serialization.Serializable

@Serializable
private object HomeRoute

@Serializable
private data class DetailRoute(val id: String, val direction: String)

@Composable
fun App(apiBaseUrl: String) {
    val container = remember(apiBaseUrl) { AppContainer(apiBaseUrl) }
    val nav = rememberNavController()

    AppTheme {
        NavHost(navController = nav, startDestination = HomeRoute) {
            composable<HomeRoute> {
                HomeScreen(
                    viewModel = viewModel { HomeViewModel(container.getCrossings) },
                    onOpen = { id, dir -> nav.navigate(DetailRoute(id, dir.name)) },
                )
            }
            composable<DetailRoute> { entry ->
                val route = entry.toRoute<DetailRoute>()
                DetailScreen(
                    viewModel = viewModel {
                        DetailViewModel(route.id, Direction.valueOf(route.direction), container.getCrossingDetail)
                    },
                    onBack = { nav.popBackStack() },
                )
            }
        }
    }
}
