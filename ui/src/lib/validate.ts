/**
 * Service Monitor Target Validation
 * Matches backend validation logic in internal/api/handlers/service_monitor_handler.go
 */

export type ProbeType = 'HTTP' | 'TCP' | 'ICMP'

export interface ValidationResult {
  valid: boolean
  cleanedValue: string
  error?: string
}

/**
 * Clean and validate monitoring target based on probe type
 */
export function validateAndCleanTarget(
  target: string,
  probeType: ProbeType
): ValidationResult {
  // Trim all whitespace including spaces, tabs, newlines
  let cleaned = target.trim()

  // Remove control characters
  cleaned = cleaned.replace(/[\x00-\x1F\x7F]/g, '')
  cleaned = cleaned.trim() // Final trim

  if (cleaned === '') {
    return {
      valid: false,
      cleanedValue: '',
      error: '目标不能为空',
    }
  }

  switch (probeType) {
    case 'HTTP':
      return validateHTTPTarget(cleaned)
    case 'ICMP':
      return validateICMPTarget(cleaned)
    case 'TCP':
      return validateTCPTarget(cleaned)
    default:
      return {
        valid: false,
        cleanedValue: '',
        error: `不支持的探测类型: ${probeType}`,
      }
  }
}

/**
 * Validate HTTP/HTTPS URL format
 */
function validateHTTPTarget(target: string): ValidationResult {
  // Add http:// prefix if missing
  let url = target
  if (!url.startsWith('http://') && !url.startsWith('https://')) {
    url = 'http://' + url
  }

  try {
    const parsed = new URL(url)

    // Validate scheme
    if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') {
      return {
        valid: false,
        cleanedValue: '',
        error: 'URL协议必须是http或https',
      }
    }

    // Validate host
    if (!parsed.hostname) {
      return {
        valid: false,
        cleanedValue: '',
        error: 'URL必须包含有效的主机名',
      }
    }

    return {
      valid: true,
      cleanedValue: parsed.toString(),
    }
  } catch (e) {
    return {
      valid: false,
      cleanedValue: '',
      error: 'URL格式无效',
    }
  }
}

/**
 * Validate IP address or domain name for ICMP ping
 */
function validateICMPTarget(target: string): ValidationResult {
  // Check if it's a valid IP address (IPv4 or IPv6)
  if (isValidIP(target)) {
    return {
      valid: true,
      cleanedValue: target,
    }
  }

  // Check if it's a valid domain name (e.g., example.com)
  const domainRegex =
    /^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$/
  if (domainRegex.test(target)) {
    return {
      valid: true,
      cleanedValue: target,
    }
  }

  // Also allow simple hostnames (no dots, e.g., localhost)
  const hostnameRegex = /^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?$/
  if (hostnameRegex.test(target)) {
    return {
      valid: true,
      cleanedValue: target,
    }
  }

  return {
    valid: false,
    cleanedValue: '',
    error: '目标必须是有效的IP地址或域名',
  }
}

/**
 * Validate TCP target in format "host:port" or "host" (port optional)
 */
function validateTCPTarget(target: string): ValidationResult {
  // Try to split host and port
  const parts = target.split(':')

  if (parts.length === 1) {
    // No port specified - validate as host only
    if (isValidHost(parts[0])) {
      return {
        valid: true,
        cleanedValue: target,
      }
    }
    return {
      valid: false,
      cleanedValue: '',
      error: 'TCP目标格式无效（期望 主机:端口 或 主机）',
    }
  }

  if (parts.length === 2) {
    const [host, portStr] = parts

    // Validate host
    if (!isValidHost(host)) {
      return {
        valid: false,
        cleanedValue: '',
        error: 'TCP目标中的主机名无效',
      }
    }

    // Validate port
    const port = parseInt(portStr, 10)
    if (isNaN(port) || port < 1 || port > 65535) {
      return {
        valid: false,
        cleanedValue: '',
        error: '端口必须在1到65535之间',
      }
    }

    return {
      valid: true,
      cleanedValue: `${host}:${port}`,
    }
  }

  // Handle IPv6 addresses with port (e.g., [::1]:8080)
  if (target.startsWith('[')) {
    const match = target.match(/^\[(.+?)\]:(\d+)$/)
    if (match) {
      const [, host, portStr] = match
      if (isValidIP(host)) {
        const port = parseInt(portStr, 10)
        if (!isNaN(port) && port >= 1 && port <= 65535) {
          return {
            valid: true,
            cleanedValue: target,
          }
        }
      }
    }
  }

  return {
    valid: false,
    cleanedValue: '',
    error: 'TCP目标格式无效',
  }
}

/**
 * Check if a string is a valid IP address (IPv4 or IPv6)
 */
function isValidIP(ip: string): boolean {
  // IPv4 validation
  const ipv4Regex = /^(\d{1,3}\.){3}\d{1,3}$/
  if (ipv4Regex.test(ip)) {
    const parts = ip.split('.')
    return parts.every((part) => {
      const num = parseInt(part, 10)
      return num >= 0 && num <= 255
    })
  }

  // IPv6 validation (simplified)
  const ipv6Regex = /^([0-9a-fA-F]{0,4}:){2,7}[0-9a-fA-F]{0,4}$/
  return ipv6Regex.test(ip)
}

/**
 * Check if a string is a valid host (IP or domain name)
 */
function isValidHost(host: string): boolean {
  // Check if it's a valid IP address
  if (isValidIP(host)) {
    return true
  }

  // Check if it's a valid domain name
  const domainRegex =
    /^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$/
  if (domainRegex.test(host)) {
    return true
  }

  // Also allow simple hostnames (no dots)
  const hostnameRegex = /^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?$/
  return hostnameRegex.test(host)
}
