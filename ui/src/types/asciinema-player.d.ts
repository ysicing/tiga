declare module 'asciinema-player' {
  export function create(
    source: { data: any; parser: string } | string,
    element: HTMLElement,
    options?: {
      fit?: 'width' | 'height' | 'both' | false;
      terminalFontSize?: string;
      pauseOnMarkers?: boolean;
      loop?: boolean;
      autoPlay?: boolean;
      [key: string]: any;
    }
  ): void;
}