import React from 'react'
import { PhotoProvider, PhotoView } from 'react-photo-view'

import 'react-photo-view/dist/react-photo-view.css'

import Editor from '@monaco-editor/react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'

type Props = {
  url: string
  filename?: string
  mime?: string
}

export const FilePreviewer: React.FC<Props> = ({ url, filename }) => {
  const lower = (filename || '').toLowerCase()
  const isImage = /\.(png|jpg|jpeg|gif|webp|bmp|svg)$/.test(lower)
  const isVideo = /\.(mp4|webm|ogg|mov)$/.test(lower)
  const isMarkdown = /\.(md|markdown)$/.test(lower)
  const isCode =
    /\.(ts|tsx|js|json|yaml|yml|go|py|java|rb|cs|cpp|c|rs|sh|txt)$/.test(lower)

  if (isImage) {
    return (
      <PhotoProvider>
        <PhotoView src={url}>
          <img src={url} alt={filename} className="max-w-full max-h-[80vh]" />
        </PhotoView>
      </PhotoProvider>
    )
  }

  if (isVideo) {
    return <video src={url} controls className="w-full max-h-[80vh]" />
  }

  if (isMarkdown) {
    return (
      <div className="prose dark:prose-invert max-w-none">
        <ReactMarkdown
          remarkPlugins={[remarkGfm]}
        >{`Loading markdown from ${url} is not supported directly here. Download and render.`}</ReactMarkdown>
      </div>
    )
  }

  if (isCode) {
    return (
      <div className="h-[70vh]">
        <Editor
          path={filename}
          defaultLanguage="plaintext"
          theme="vs-dark"
          options={{ readOnly: true }}
          value={`// Preview: ${filename}\n// Open in a new tab: ${url}`}
        />
      </div>
    )
  }

  return (
    <div>
      <a href={url} target="_blank" rel="noreferrer" className="underline">
        Open file
      </a>
    </div>
  )
}
