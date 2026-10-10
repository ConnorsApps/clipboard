import { UploadIcon } from './icons'

function DropOverlay() {
  return (
    <div className="fixed inset-0 z-50 p-4 bg-dark-950/80 backdrop-blur-sm pointer-events-none">
      <div className="h-full rounded-3xl border-2 border-dashed border-accent-500 flex flex-col items-center justify-center gap-3 text-white">
        <UploadIcon className="w-12 h-12 text-accent-500" />
        <p className="text-xl font-semibold">Drop to upload</p>
      </div>
    </div>
  )
}

export default DropOverlay
