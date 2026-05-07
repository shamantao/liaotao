/*
 * ProjectModels.kt - project domain entities for Liaotao.
 * Responsibilities: define project identity and lifecycle metadata used by
 * services and repositories across the domain.
 */

package io.liaotao.domain.projects

import java.time.Instant

data class Project(
    val id: String,
    val name: String,
    val description: String,
    val createdAt: Instant,
    val updatedAt: Instant,
)

data class CreateProjectRequest(
    val name: String,
    val description: String = "",
)

data class UpdateProjectRequest(
    val name: String,
    val description: String = "",
)