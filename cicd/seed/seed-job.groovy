// cicd/seed/seed-job.groovy
// =================================================================
// Jenkins Job DSL — Seed Job
// Run this ONCE in Jenkins to auto-create all 15 service pipelines.
// Each job:
//   - Watches Git for changes in its service folder OR pkg/
//   - Points to cicd/Jenkinsfile (NOT a per-service Jenkinsfile)
//   - Passes SERVICE_NAME as default parameter
// =================================================================

// All service definitions — mirrors services.yaml
def SERVICES = [
    [name: 'api-gateway',   watchPaths: ['api-gateway/**', 'pkg/**', 'cicd/**']],
    [name: 'auth',          watchPaths: ['auth/**',        'pkg/**', 'cicd/**']],
    [name: 'profile',       watchPaths: ['profile/**',     'pkg/**', 'cicd/**']],
    [name: 'catalog',       watchPaths: ['catalog/**',     'pkg/**', 'cicd/**']],
    [name: 'cart',          watchPaths: ['cart/**',        'pkg/**', 'cicd/**']],
    [name: 'order',         watchPaths: ['order/**',       'pkg/**', 'cicd/**']],
    [name: 'inventory',     watchPaths: ['inventory/**',   'pkg/**', 'cicd/**']],
    [name: 'payment',       watchPaths: ['payment/**',     'pkg/**', 'cicd/**']],
    [name: 'logistic',      watchPaths: ['logistic/**',    'pkg/**', 'cicd/**']],
    [name: 'notification',  watchPaths: ['notification/**','pkg/**', 'cicd/**']],
    [name: 'media',         watchPaths: ['media/**',       'pkg/**', 'cicd/**']],
    [name: 'search',        watchPaths: ['search/**',      'pkg/**', 'cicd/**']],
    [name: 'campaign',      watchPaths: ['campaign/**',    'pkg/**', 'cicd/**']],
    [name: 'review',        watchPaths: ['review/**',      'pkg/**', 'cicd/**']],
    [name: 'analytic',      watchPaths: ['analytic/**',    'pkg/**', 'cicd/**']],
]

def GIT_REPO_URL   = binding.variables['GIT_REPO_URL']    // Jenkins global variable
def GIT_CREDS_ID   = binding.variables['GIT_CREDS_ID'] ?: 'git.credentials'
def JENKINS_FOLDER = 'TAFU'

// Create a folder to group all TAFU pipelines
folder(JENKINS_FOLDER) {
    displayName('TAFU Microservices')
    description('All CI/CD pipelines for TAFU microservices. Managed by seed job.')
}

// Create one pipeline job per service
SERVICES.each { svc ->
    pipelineJob("${JENKINS_FOLDER}/${svc.name}") {
        displayName("tafu — ${svc.name}")
        description("""
            CI/CD pipeline for the '${svc.name}' service.
            Triggers on changes to: ${svc.watchPaths.join(', ')}
            Uses shared Jenkinsfile: cicd/Jenkinsfile
        """.stripIndent().trim())

        // ── Default parameters (service-specific values passed into cicd/Jenkinsfile)
        parameters {
            stringParam(
                'SERVICE_NAME',
                svc.name,                       // ← Default = this service's name
                "Service to build. DO NOT change this manually."
            )
            choiceParam(
                'FORCE_DEPLOY_ENV',
                ['auto', 'staging', 'production', 'none'],
                "Override deploy target. Use 'auto' for branch-based deploy."
            )
        }

        // ── Pipeline source: cicd/Jenkinsfile (shared, not per-service)
        definition {
            cpsScm {
                scm {
                    git {
                        remote {
                            url(GIT_REPO_URL)
                            credentials(GIT_CREDS_ID)
                        }
                        branches('*/main', '*/develop', '*/feature/*')
                    }
                }
                scriptPath('cicd/Jenkinsfile')   // ← Points to the ONE shared Jenkinsfile
                lightweight(true)
            }
        }

        // ── Git webhook trigger with path filtering
        triggers {
            scm('H/5 * * * *')   // Poll every 5 min as fallback (webhook is primary)

            // Path-based trigger: only run when these paths have changes
            // Requires "Generic Webhook Trigger" or "GitHub/GitLab" plugin
            // Configure webhook path filter in Jenkins job settings after creation
        }

        // ── Build rotation
        logRotator {
            numToKeep(20)
            artifactNumToKeep(5)
        }

        // ── Prevent concurrent builds for the same service
        concurrentBuild(false)
    }
}

// ── Bonus: Shared packages validation job (no deploy)
pipelineJob("${JENKINS_FOLDER}/pkg-validation") {
    displayName("tafu — pkg/ (shared packages)")
    description("Validates all shared packages on every change. No deploy step.")

    parameters {
        stringParam('SERVICE_NAME', 'pkg', 'Used for identification only.')
    }

    definition {
        cpsScm {
            scm {
                git {
                    remote { url(GIT_REPO_URL); credentials(GIT_CREDS_ID) }
                    branches('*/main', '*/develop')
                }
            }
            scriptPath('cicd/Jenkinsfile.pkg')
            lightweight(true)
        }
    }

    logRotator { numToKeep(10) }
    concurrentBuild(false)
}
