import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { envService } from '../services';
import { toast } from 'sonner';
import { Save, RefreshCw } from 'lucide-react';

// TAREA 3: Component for editing .env file
export function DotenvEditor() {
  const queryClient = useQueryClient();
  const [content, setContent] = useState('');
  const [isEditing, setIsEditing] = useState(false);

  const { isLoading: isLoadingContent } = useQuery({
    queryKey: ['dotenv'],
    queryFn: async () => {
      const data = await envService.getDotenv();
      setContent(data.content);
      return data;
    },
  });

  const saveMutation = useMutation({
    mutationFn: () => envService.saveDotenv(content),
    onSuccess: (data) => {
      toast.success(data.message);
      setIsEditing(false);
      if (data.restarted) {
        toast.info('Node-RED se está reiniciando...');
      }
    },
    onError: (error: unknown) => {
      let message = 'Error al guardar .env';
      if (error && typeof error === 'object' && 'response' in error) {
        const err = error as { response?: { data?: { error?: { message?: string } } } };
        if (err.response?.data?.error?.message) {
          message = err.response.data.error.message;
        }
      }
      toast.error(message);
    },
  });

  const handleSave = () => {
    saveMutation.mutate();
  };

  const handleCancel = () => {
    // Reload from server
    queryClient.invalidateQueries({ queryKey: ['dotenv'] });
    setIsEditing(false);
  };

  return (
    <div className="space-y-4">
      <div className="flex justify-between items-start">
        <div>
          <h2 className="text-lg font-semibold text-base-content">Archivo .env</h2>
          <p className="text-sm text-base-content/60 mt-1">
            Las variables aquí tienen prioridad sobre las configuradas en la tabla. Node-RED se reiniciará automáticamente al guardar.
          </p>
        </div>
      </div>

      <div className="surface-card p-4">
        {isLoadingContent ? (
          <div className="flex items-center justify-center py-8">
            <div className="loading loading-spinner loading-sm"></div>
          </div>
        ) : (
          <>
            <textarea
              value={content}
              onChange={(e) => {
                setContent(e.target.value);
                setIsEditing(true);
              }}
              placeholder={`# Variables de entorno para Node-RED
# Formato: CLAVE=VALOR
# Las líneas con # son comentarios

# Base de datos
# DB_HOST=localhost
# DB_PORT=5432

# MQTT
# MQTT_USER=admin
# MQTT_PASS=secret`}
              className="w-full h-64 p-3 font-mono text-sm bg-base-100 border border-border rounded resize-none focus:outline-none focus:border-primary focus:ring-2 focus:ring-primary/20"
            />

            <div className="flex gap-2 mt-4">
              <button
                onClick={handleSave}
                disabled={!isEditing || saveMutation.isPending}
                className="action-btn-primary gap-2 flex items-center disabled:opacity-50 disabled:cursor-not-allowed"
              >
                <Save className="w-4 h-4" />
                {saveMutation.isPending ? (
                  <>
                    <RefreshCw className="w-4 h-4 animate-spin" />
                    Guardando...
                  </>
                ) : (
                  'Guardar .env'
                )}
              </button>
              {isEditing && (
                <button
                  onClick={handleCancel}
                  disabled={saveMutation.isPending}
                  className="action-btn-secondary disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  Cancelar
                </button>
              )}
            </div>
          </>
        )}
      </div>
    </div>
  );
}

export default DotenvEditor;
