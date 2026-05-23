import { useState, useRef, useEffect } from 'react';
import { Upload, X, Image as ImageIcon, Loader2 } from 'lucide-react';
import { toast } from 'sonner';
import { fileService } from '@/features/configuration/services';
import { cn } from '@/shared/lib';

interface ImageUploadProps {
  label: string;
  value: string;
  onChange: (url: string) => void;
  type: 'favicon' | 'header' | 'login';
  help?: string;
}

export function ImageUpload({ label, value, onChange, type, help }: ImageUploadProps) {
  const [isUploading, setIsUploading] = useState(false);
  const [preview, setPreview] = useState<string | null>(value || null);
  const inputRef = useRef<HTMLInputElement>(null);

  // Sincroniza el preview cuando el value llega desde la config cargada
  useEffect(() => {
    setPreview(value || null);
  }, [value]);

  const handleFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    // Validate file type
    if (!file.type.startsWith('image/')) {
      toast.error('Please select an image file (PNG, JPG, GIF, etc.)');
      return;
    }

    // Validate file size (max 2MB)
    if (file.size > 2 * 1024 * 1024) {
      const sizeMB = (file.size / 1024 / 1024).toFixed(2);
      toast.error(`Image must be less than 2MB (file is ${sizeMB}MB)`);
      return;
    }

    // Validate dimensions for specific types
    if (['favicon'].includes(type) && file.type.startsWith('image/')) {
      // Optional: could add dimension validation later
    }

    setIsUploading(true);

    try {
      const response = await fileService.uploadImage(type, file);
      
      if (response.data.success && response.data.data) {
        const { url, filename } = response.data.data;
        setPreview(url);
        onChange(url);
        toast.success(`Uploaded ${filename}`);
      } else {
        const errorMsg = response.data.error?.message || 'Failed to upload image';
        toast.error(errorMsg);
      }
    } catch (error: unknown) {
      console.error('Upload error:', error);
      if (error instanceof Error) {
        toast.error(`Upload failed: ${error.message}`);
      } else {
        toast.error('Failed to upload image. Please try again.');
      }
    } finally {
      setIsUploading(false);
      // Reset file input
      if (inputRef.current) {
        inputRef.current.value = '';
      }
    }
  };

  const handleRemove = () => {
    setPreview(null);
    onChange('');
    if (inputRef.current) {
      inputRef.current.value = '';
    }
  };

  return (
    <div className="space-y-2">
      <label className="text-sm font-medium text-base-content">{label}</label>
      
      {preview ? (
        <div className="relative group overflow-hidden rounded-2xl border border-border">
          <img
            src={preview}
            alt={label}
            className={cn(
              "w-full object-cover",
              type === 'favicon' ? 'h-16 w-16' : 'h-32 w-full'
            )}
          />
          <div className="image-overlay">
            <button
              type="button"
              onClick={() => inputRef.current?.click()}
              className="image-overlay-btn"
            >
              <Upload className="w-4 h-4" />
            </button>
            <button
              type="button"
              onClick={handleRemove}
              className="image-overlay-btn hover:!bg-error/80"
            >
              <X className="w-4 h-4" />
            </button>
          </div>
        </div>
      ) : (
        <button
          type="button"
          onClick={() => inputRef.current?.click()}
          disabled={isUploading}
          className={cn(
            "h-24 w-full rounded-2xl border-2 border-dashed border-border",
            "flex flex-col items-center justify-center gap-2",
            "text-base-content/55 hover:border-primary/50 hover:text-base-content",
            "transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          )}
        >
          {isUploading ? (
            <>
              <Loader2 className="w-6 h-6 animate-spin" />
              <span className="text-sm">Uploading...</span>
            </>
          ) : (
            <>
              <ImageIcon className="w-6 h-6" />
              <span className="text-sm">Click to upload image</span>
            </>
          )}
        </button>
      )}
      
      <input
        ref={inputRef}
        type="file"
        accept="image/*"
        onChange={handleFileChange}
        className="hidden"
      />
      
      {help && <p className="text-xs text-base-content/60">{help}</p>}
    </div>
  );
}
