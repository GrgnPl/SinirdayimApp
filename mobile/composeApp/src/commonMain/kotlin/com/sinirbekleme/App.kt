package com.sinirbekleme

import androidx.compose.foundation.layout.consumeWindowInsets
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Route
import androidx.compose.material.icons.rounded.Traffic
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.NavigationBarItemDefaults
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.unit.dp
import androidx.lifecycle.viewmodel.compose.viewModel
import androidx.navigation.NavDestination.Companion.hasRoute
import androidx.navigation.NavGraph.Companion.findStartDestination
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.currentBackStackEntryAsState
import androidx.navigation.compose.rememberNavController
import androidx.navigation.toRoute
import com.sinirbekleme.di.AppContainer
import com.sinirbekleme.domain.model.Direction
import com.sinirbekleme.presentation.detail.DetailScreen
import com.sinirbekleme.presentation.detail.DetailViewModel
import com.sinirbekleme.presentation.home.HomeScreen
import com.sinirbekleme.presentation.home.HomeViewModel
import com.sinirbekleme.presentation.theme.AppTheme
import com.sinirbekleme.presentation.trip.TripFormScreen
import com.sinirbekleme.presentation.trip.TripResultScreen
import com.sinirbekleme.presentation.trip.TripViewModel
import kotlinx.serialization.Serializable

@Serializable
private object HomeRoute

@Serializable
private data class DetailRoute(val id: String, val direction: String)

@Serializable
private object TripRoute

@Serializable
private object TripResultRoute

private data class Tab(val route: Any, val label: String, val icon: ImageVector)

private val tabs = listOf(
    Tab(HomeRoute, "Kapılar", Icons.Rounded.Traffic),
    Tab(TripRoute, "Rota", Icons.Rounded.Route),
)

@Composable
fun App(apiBaseUrl: String) {
    val container = remember(apiBaseUrl) { AppContainer(apiBaseUrl) }
    val nav = rememberNavController()
    // Shared by the trip form and result screens.
    val tripViewModel = viewModel { TripViewModel(container.searchPlaces, container.planTrip) }

    AppTheme {
        val entry by nav.currentBackStackEntryAsState()
        val destination = entry?.destination
        val onTab = destination?.let { d -> d.hasRoute<HomeRoute>() || d.hasRoute<TripRoute>() } ?: true

        Scaffold(
            containerColor = MaterialTheme.colorScheme.background,
            bottomBar = {
                if (onTab) {
                    NavigationBar(containerColor = MaterialTheme.colorScheme.surface, tonalElevation = 0.dp) {
                        tabs.forEach { tab ->
                            val selected = destination?.hasRoute(tab.route::class) == true
                            NavigationBarItem(
                                selected = selected,
                                onClick = {
                                    nav.navigate(tab.route) {
                                        popUpTo(nav.graph.findStartDestination().id) { saveState = true }
                                        launchSingleTop = true
                                        restoreState = true
                                    }
                                },
                                icon = { Icon(tab.icon, contentDescription = null) },
                                label = { Text(tab.label) },
                                colors = NavigationBarItemDefaults.colors(
                                    indicatorColor = MaterialTheme.colorScheme.surfaceVariant,
                                ),
                            )
                        }
                    }
                }
            },
        ) { inner ->
            NavHost(
                navController = nav,
                startDestination = HomeRoute,
                modifier = Modifier.padding(inner).consumeWindowInsets(inner),
            ) {
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
                composable<TripRoute> {
                    TripFormScreen(tripViewModel, onPlanned = { nav.navigate(TripResultRoute) })
                }
                composable<TripResultRoute> {
                    TripResultScreen(tripViewModel, onBack = { nav.popBackStack() })
                }
            }
        }
    }
}
