import { useEffect, useRef, useState } from 'react'
import * as AsciinemaPlayerLib from 'asciinema-player'
import 'asciinema-player/dist/bundle/asciinema-player.css'

export interface AsciinemaPlayerProps {
  /** Recording content in Asciinema v2 format */
  content: string
  /** Player options */
  options?: {
    /** Auto play on mount */
    autoPlay?: boolean
    /** Loop playback */
    loop?: boolean
    /** Start at specific time (seconds) */
    startAt?: number
    /** Playback speed */
    speed?: number
    /** Idle time limit (seconds) */
    idleTimeLimit?: number
    /** Player theme */
    theme?: 'asciinema' | 'tango' | 'solarized-dark' | 'solarized-light' | 'monokai'
    /** Show poster before playback */
    poster?: string
    /** Fit to container */
    fit?: 'width' | 'height' | 'both' | 'none'
    /** Terminal font size */
    terminalFontSize?: string
    /** Terminal font family */
    terminalFontFamily?: string
    /** Terminal line height */
    terminalLineHeight?: number
  }
  /** Player container class name */
  className?: string
  /** Error callback */
  onError?: (error: Error) => void
}

/**
 * Asciinema Player React Component
 *
 * Wraps asciinema-player for React usage. Supports standard Asciinema v2 format.
 *
 * @example
 * ```tsx
 * <AsciinemaPlayer
 *   content={recordingContent}
 *   options={{
 *     autoPlay: false,
 *     loop: false,
 *     speed: 1.0,
 *     theme: 'asciinema',
 *   }}
 * />
 * ```
 */
export function AsciinemaPlayer({
  content,
  options = {},
  className = '',
  onError,
}: AsciinemaPlayerProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const playerRef = useRef<any>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!containerRef.current || !content) return

    try {
      // Clear previous player instance
      if (playerRef.current) {
        containerRef.current.innerHTML = ''
        playerRef.current = null
      }

      // Default player options
      const playerOptions: any = {
        autoPlay: options.autoPlay ?? false,
        loop: options.loop ?? false,
        startAt: options.startAt ?? 0,
        speed: options.speed ?? 1.0,
        idleTimeLimit: options.idleTimeLimit,
        theme: options.theme ?? 'asciinema',
        poster: options.poster,
        fit: options.fit ?? 'width',
        terminalFontSize: options.terminalFontSize ?? '15px',
        terminalFontFamily: options.terminalFontFamily ?? 'Monaco, monospace',
        terminalLineHeight: options.terminalLineHeight ?? 1.33,
        cols: undefined, // Auto-detect from recording
        rows: undefined, // Auto-detect from recording
        controls: false, // Hide controls
      }

      // Create player from text content
      // asciinema-player expects a URL, so we convert text content to a data URL
      const dataUrl = `data:application/json;charset=utf-8,${encodeURIComponent(content)}`
      playerRef.current = AsciinemaPlayerLib.create(
        dataUrl, // Data URL with Asciinema content
        containerRef.current,
        playerOptions
      )

      setError(null)
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to create player'
      setError(errorMessage)
      if (onError) {
        onError(err instanceof Error ? err : new Error(errorMessage))
      }
      console.error('Asciinema player error:', err)
    }

    // Cleanup on unmount
    return () => {
      if (containerRef.current) {
        containerRef.current.innerHTML = ''
      }
      playerRef.current = null
    }
  }, [content, options, onError])

  if (error) {
    return (
      <div className="rounded-lg border border-destructive bg-destructive/10 p-6 text-center">
        <p className="text-destructive font-medium mb-2">播放器加载失败</p>
        <p className="text-sm text-muted-foreground">{error}</p>
      </div>
    )
  }

  return (
    <div
      ref={containerRef}
      className={`asciinema-player-container ${className}`}
      style={{
        // Ensure proper sizing
        width: '100%',
        maxWidth: '100%',
      }}
    />
  )
}

/**
 * Utility function to validate Asciinema v2 format
 */
export function validateAsciinemaContent(content: string): boolean {
  try {
    const lines = content.trim().split('\n')
    if (lines.length < 2) return false

    // First line should be header (JSON)
    const header = JSON.parse(lines[0])
    if (header.version !== 2) return false

    // Second line onwards should be events [time, type, data]
    for (let i = 1; i < lines.length; i++) {
      const event = JSON.parse(lines[i])
      if (!Array.isArray(event) || event.length !== 3) return false
      if (typeof event[0] !== 'number') return false
      if (!['o', 'i'].includes(event[1])) return false
      if (typeof event[2] !== 'string') return false
    }

    return true
  } catch {
    return false
  }
}
