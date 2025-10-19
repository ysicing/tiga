import { useTranslation } from 'react-i18next'
import { Link, useLocation } from 'react-router-dom'

import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from '@/components/ui/breadcrumb'

interface BreadcrumbSegment {
  label: string
  href?: string
}

export function DynamicBreadcrumb() {
  const location = useLocation()
  const { t } = useTranslation()

  const generateBreadcrumbs = (): BreadcrumbSegment[] => {
    const pathSegments = location.pathname.split('/').filter(Boolean)
    const breadcrumbs: BreadcrumbSegment[] = []

    if (pathSegments.length === 0) {
      return breadcrumbs
    }

    // Resource name mappings
    const resourceLabels: Record<string, string> = {
      pods: t('nav.pods'),
      deployments: t('nav.deployments'),
      services: t('nav.services'),
      configmaps: t('nav.configMaps'),
      secrets: t('nav.secrets'),
      ingresses: t('nav.ingresses'),
      gateways: t('nav.gateways'),
      httproutes: t('nav.httproutes'),
      jobs: t('nav.jobs'),
      daemonsets: t('nav.daemonsets'),
      statefulsets: t('nav.statefulsets'),
      namespaces: t('nav.namespaces'),
      pvcs: t('sidebar.short.pvcs'),
      crds: t('nav.crds'),
      crs: t('nav.customResources'),
      horizontalpodautoscalers: t('nav.horizontalpodautoscalers'),
    }

    // Helper function to create breadcrumb item
    const createBreadcrumb = (
      label: string,
      href?: string
    ): BreadcrumbSegment => ({
      label: resourceLabels[label] || label,
      href,
    })

    // Helper function to get safe link for segments
    const getSafeLink = (index: number): string | undefined => {
      const isLastSegment = index === pathSegments.length - 1
      if (isLastSegment) return undefined

      // Detect CRD routes: if in /k8s and second segment contains dots (e.g., clonesets.apps.kruise.io)
      const isCRDRoute = pathSegments[0] === 'k8s' && pathSegments[1]?.includes('.')

      if (isCRDRoute) {
        // For CRD routes like /k8s/clonesets.apps.kruise.io/namespace/name
        if (index === 0) return '/k8s' // K8s home
        if (index === 1) return `/k8s/${pathSegments[1]}` // CRD list (e.g., /k8s/clonesets.apps.kruise.io)
        if (index === 2) return `/k8s/${pathSegments[1]}` // namespace links back to CRD list
        return undefined
      } else if (pathSegments[0] === 'k8s') {
        // Regular K8s resources
        return `/k8s/${pathSegments.slice(1, index + 1).join('/')}`
      } else {
        // Regular resources: namespace should link back to resource list
        const isNamespace = pathSegments.length === 3 && index === 1
        if (isNamespace) return `/${pathSegments[0]}`
        return `/${pathSegments.slice(0, index + 1).join('/')}`
      }
    }

    // Generate breadcrumbs for each path segment
    pathSegments.forEach((segment, index) => {
      const href = getSafeLink(index)
      breadcrumbs.push(createBreadcrumb(segment, href))
    })

    return breadcrumbs
  }

  const breadcrumbs = generateBreadcrumbs()

  return (
    <Breadcrumb className="hidden md:block">
      <BreadcrumbList>
        {breadcrumbs.map((crumb, index) => (
          <div key={index} className="flex items-center">
            {index > 0 && <BreadcrumbSeparator />}
            <BreadcrumbItem>
              {crumb.href ? (
                <BreadcrumbLink asChild>
                  <Link to={crumb.href}>{crumb.label}</Link>
                </BreadcrumbLink>
              ) : (
                <BreadcrumbPage>{crumb.label}</BreadcrumbPage>
              )}
            </BreadcrumbItem>
          </div>
        ))}
      </BreadcrumbList>
    </Breadcrumb>
  )
}
