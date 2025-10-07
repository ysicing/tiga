import { useState } from 'react'
import yaml from 'js-yaml'
import { Deployment } from 'kubernetes-types/apps/v1'
import { Container } from 'kubernetes-types/core/v1'
import { Plus, Trash2, X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { createResource } from '@/lib/api'
import { translateError } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Separator } from '@/components/ui/separator'

import { NamespaceSelector } from '../selector/namespace-selector'
import { SimpleYamlEditor } from '../simple-yaml-editor'
import { EnvironmentEditor } from './environment-editor'
import { ImageEditor } from './image-editor'

interface DeploymentCreateDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSuccess: (deployment: Deployment, namespace: string) => void
  defaultNamespace?: string
}

interface ContainerConfig {
  name: string
  image: string
  port?: number
  pullPolicy: 'Always' | 'IfNotPresent' | 'Never'
  resources: {
    requests: {
      cpu: string
      memory: string
    }
    limits: {
      cpu: string
      memory: string
    }
  }
  container: Container
}

interface DeploymentFormData {
  name: string
  namespace: string
  replicas: number
  labels: Array<{ key: string; value: string }>
  containers: ContainerConfig[]
}

const createDefaultContainer = (index: number): ContainerConfig => ({
  name: `container-${index + 1}`,
  image: '',
  pullPolicy: 'IfNotPresent',
  resources: {
    requests: {
      cpu: '',
      memory: '',
    },
    limits: {
      cpu: '',
      memory: '',
    },
  },
  container: {
    name: `container-${index + 1}`,
    image: '',
  },
})

const initialFormData: DeploymentFormData = {
  name: '',
  namespace: 'default',
  replicas: 1,
  labels: [{ key: 'app', value: '' }],
  containers: [createDefaultContainer(0)],
}

export function DeploymentCreateDialog({
  open,
  onOpenChange,
  onSuccess,
  defaultNamespace,
}: DeploymentCreateDialogProps) {
  const [formData, setFormData] = useState<DeploymentFormData>({
    ...initialFormData,
    namespace: defaultNamespace || 'default',
  })
  const [isCreating, setIsCreating] = useState(false)
  const [step, setStep] = useState(1)
  const [editedYaml, setEditedYaml] = useState<string>('')
  const { t } = useTranslation()
  const totalSteps = 3

  const updateFormData = (updates: Partial<DeploymentFormData>) => {
    setFormData((prev) => ({ ...prev, ...updates }))
  }

  const addLabel = () => {
    setFormData((prev) => ({
      ...prev,
      labels: [...prev.labels, { key: '', value: '' }],
    }))
  }

  const updateLabel = (
    index: number,
    field: 'key' | 'value',
    value: string
  ) => {
    setFormData((prev) => ({
      ...prev,
      labels: prev.labels.map((label, i) =>
        i === index ? { ...label, [field]: value } : label
      ),
    }))
  }

  const removeLabel = (index: number) => {
    setFormData((prev) => ({
      ...prev,
      labels: prev.labels.filter((_, i) => i !== index),
    }))
  }

  const addContainer = () => {
    setFormData((prev) => ({
      ...prev,
      containers: [
        ...prev.containers,
        createDefaultContainer(prev.containers.length),
      ],
    }))
  }

  const removeContainer = (index: number) => {
    if (formData.containers.length <= 1) {
      toast.error('At least one container is required')
      return
    }
    setFormData((prev) => ({
      ...prev,
      containers: prev.containers.filter((_, i) => i !== index),
    }))
  }

  const updateContainer = (
    index: number,
    updates: Partial<ContainerConfig>
  ) => {
    setFormData((prev) => ({
      ...prev,
      containers: prev.containers.map((container, i) =>
        i === index ? { ...container, ...updates } : container
      ),
    }))
  }

  const generateDeploymentYaml = (): string => {
    // Build deployment object
    const labelsObj = formData.labels.reduce(
      (acc, label) => {
        if (label.key && label.value) {
          acc[label.key] = label.value
        }
        return acc
      },
      {} as Record<string, string>
    )

    // Ensure app label matches name if not set
    if (!labelsObj.app && formData.name) {
      labelsObj.app = formData.name
    }

    // Build containers array
    const containers = formData.containers.map((containerConfig) => {
      const container: Container = {
        name: containerConfig.name,
        image: containerConfig.image,
        imagePullPolicy: containerConfig.pullPolicy,
        ...(containerConfig.container.env &&
          containerConfig.container.env.length > 0 && {
            env: containerConfig.container.env.filter(
              (env) => env.name && (env.value || env.valueFrom)
            ),
          }),
        ...(containerConfig.container.envFrom &&
          containerConfig.container.envFrom.length > 0 && {
            envFrom: containerConfig.container.envFrom.filter(
              (source) => source.configMapRef?.name || source.secretRef?.name
            ),
          }),
        ...(containerConfig.port && {
          ports: [
            {
              containerPort: containerConfig.port,
            },
          ],
        }),
        ...((containerConfig.resources.requests.cpu ||
          containerConfig.resources.requests.memory ||
          containerConfig.resources.limits.cpu ||
          containerConfig.resources.limits.memory) && {
          resources: {
            ...((containerConfig.resources.requests.cpu ||
              containerConfig.resources.requests.memory) && {
              requests: {
                ...(containerConfig.resources.requests.cpu && {
                  cpu: containerConfig.resources.requests.cpu,
                }),
                ...(containerConfig.resources.requests.memory && {
                  memory: containerConfig.resources.requests.memory,
                }),
              },
            }),
            ...((containerConfig.resources.limits.cpu ||
              containerConfig.resources.limits.memory) && {
              limits: {
                ...(containerConfig.resources.limits.cpu && {
                  cpu: containerConfig.resources.limits.cpu,
                }),
                ...(containerConfig.resources.limits.memory && {
                  memory: containerConfig.resources.limits.memory,
                }),
              },
            }),
          },
        }),
      }
      return container
    })

    const deployment: Deployment = {
      apiVersion: 'apps/v1',
      kind: 'Deployment',
      metadata: {
        name: formData.name,
        namespace: formData.namespace,
        labels: labelsObj,
      },
      spec: {
        replicas: formData.replicas,
        selector: {
          matchLabels: labelsObj,
        },
        template: {
          metadata: {
            labels: labelsObj,
          },
          spec: {
            containers,
          },
        },
      },
    }

    return yaml.dump(deployment, { indent: 2, noRefs: true })
  }

  const validateStep = (stepNum: number): boolean => {
    switch (stepNum) {
      case 1:
        return !!(
          formData.name &&
          formData.namespace &&
          formData.replicas > 0 &&
          formData.labels.every((label) => label.key && label.value)
        )
      case 2:
        return formData.containers.every(
          (container) => container.image && container.name
        )
      case 3:
        return true // Review step - always valid
      default:
        return true
    }
  }

  const handleNext = () => {
    if (validateStep(step)) {
      setStep((prev) => Math.min(prev + 1, totalSteps))
    }
  }

  const handlePrevious = () => {
    setStep((prev) => Math.max(prev - 1, 1))
  }

  const handleCreate = async () => {
    if (!validateStep(step)) return

    setIsCreating(true)
    try {
      // Parse the edited YAML
      let deployment: Deployment
      try {
        const yamlContent = editedYaml || generateDeploymentYaml()
        deployment = yaml.load(yamlContent) as Deployment
      } catch (yamlError) {
        console.error('Failed to parse YAML:', yamlError)
        toast.error(
          `Invalid YAML format: ${
            yamlError instanceof Error ? yamlError.message : 'Unknown error'
          }`
        )
        return
      }

      // Validate required fields
      if (!deployment.metadata?.name || !deployment.metadata?.namespace) {
        toast.error('Deployment must have a name and namespace')
        return
      }

      const createdDeployment = await createResource(
        'deployments',
        deployment.metadata.namespace,
        deployment
      )

      toast.success(
        `Deployment "${deployment.metadata.name}" created successfully in namespace "${deployment.metadata.namespace}"`
      )

      // Reset form and close dialog
      setFormData({
        ...initialFormData,
        namespace: defaultNamespace || 'default',
      })
      setStep(1)
      setEditedYaml('')
      onOpenChange(false)

      // Call success callback with created deployment
      onSuccess(createdDeployment, deployment.metadata.namespace)
    } catch (error) {
      console.error('Failed to create deployment:', error)
      toast.error(translateError(error, t))
    } finally {
      setIsCreating(false)
    }
  }

  const handleDialogChange = (open: boolean) => {
    if (!open) {
      // Reset form when dialog closes
      setFormData({
        ...initialFormData,
        namespace: defaultNamespace || 'default',
      })
      setStep(1)
      setEditedYaml('')
    }
    onOpenChange(open)
  }

  const renderStep = () => {
    switch (step) {
      case 1:
        return (
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="name">Deployment Name *</Label>
              <Input
                id="name"
                value={formData.name}
                onChange={(e) => {
                  const value = e.target.value
                  updateFormData({
                    name: value,
                  })
                  // Update app label with full name value
                  const appLabelIndex = formData.labels.findIndex(
                    (l) => l.key === 'app'
                  )
                  if (appLabelIndex !== -1) {
                    updateLabel(appLabelIndex, 'value', value)
                  }
                }}
                placeholder="my-app"
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="namespace">Namespace *</Label>
              <NamespaceSelector
                selectedNamespace={formData.namespace}
                handleNamespaceChange={(namespace) =>
                  updateFormData({ namespace })
                }
              />
            </div>
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <Label>Labels *</Label>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={addLabel}
                >
                  <Plus className="w-4 h-4 mr-2" />
                  Add Label
                </Button>
              </div>
              <div className="space-y-2">
                {formData.labels.map((label, index) => (
                  <div key={index} className="flex gap-2 items-center">
                    <Input
                      placeholder="key"
                      value={label.key}
                      onChange={(e) =>
                        updateLabel(index, 'key', e.target.value)
                      }
                    />
                    <Input
                      placeholder="value"
                      value={label.value}
                      onChange={(e) =>
                        updateLabel(index, 'value', e.target.value)
                      }
                    />
                    {formData.labels.length > 1 && (
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        onClick={() => removeLabel(index)}
                      >
                        <X className="w-4 h-4" />
                      </Button>
                    )}
                  </div>
                ))}
              </div>
            </div>
            <div className="space-y-2">
              <Label htmlFor="replicas">Replicas *</Label>
              <Input
                id="replicas"
                type="number"
                min="1"
                value={formData.replicas}
                onChange={(e) =>
                  updateFormData({ replicas: parseInt(e.target.value) || 1 })
                }
                required
              />
            </div>
          </div>
        )

      case 2:
        return (
          <div className="space-y-6">
            <div className="flex items-center justify-between">
              <Label className="text-lg font-medium">Containers</Label>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={addContainer}
              >
                <Plus className="w-4 h-4 mr-2" />
                Add Container
              </Button>
            </div>

            {formData.containers.map((containerConfig, containerIndex) => (
              <Card key={containerIndex}>
                <CardHeader className="pb-3">
                  <div className="flex items-center justify-between">
                    <CardTitle className="text-base">
                      Container {containerIndex + 1}
                    </CardTitle>
                    {formData.containers.length > 1 && (
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        onClick={() => removeContainer(containerIndex)}
                      >
                        <Trash2 className="w-4 h-4" />
                      </Button>
                    )}
                  </div>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="grid grid-cols-2 gap-4">
                    <div className="space-y-2">
                      <ImageEditor
                        container={containerConfig.container}
                        onUpdate={(updates) =>
                          updateContainer(containerIndex, {
                            image: updates.image,
                            container: {
                              ...containerConfig.container,
                              ...updates,
                            },
                          })
                        }
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor={`name-${containerIndex}`}>
                        Container Name *
                      </Label>
                      <Input
                        id={`name-${containerIndex}`}
                        value={containerConfig.name}
                        onChange={(e) =>
                          updateContainer(containerIndex, {
                            name: e.target.value,
                            container: {
                              ...containerConfig.container,
                              name: e.target.value,
                            },
                          })
                        }
                        placeholder="container-name"
                        required
                      />
                    </div>
                  </div>

                  <div className="space-y-2">
                    <Label>Resources (optional)</Label>
                    <div className="grid grid-cols-2 gap-4">
                      <div className="space-y-2">
                        <Label className="text-sm text-muted-foreground">
                          Requests
                        </Label>
                        <div className="space-y-1">
                          <Input
                            placeholder="CPU (e.g., 100m)"
                            value={containerConfig.resources.requests.cpu}
                            onChange={(e) =>
                              updateContainer(containerIndex, {
                                resources: {
                                  ...containerConfig.resources,
                                  requests: {
                                    ...containerConfig.resources.requests,
                                    cpu: e.target.value,
                                  },
                                },
                              })
                            }
                          />
                          <Input
                            placeholder="Memory (e.g., 128Mi)"
                            value={containerConfig.resources.requests.memory}
                            onChange={(e) =>
                              updateContainer(containerIndex, {
                                resources: {
                                  ...containerConfig.resources,
                                  requests: {
                                    ...containerConfig.resources.requests,
                                    memory: e.target.value,
                                  },
                                },
                              })
                            }
                          />
                        </div>
                      </div>
                      <div className="space-y-2">
                        <Label className="text-sm text-muted-foreground">
                          Limits
                        </Label>
                        <div className="space-y-1">
                          <Input
                            placeholder="CPU (e.g., 500m)"
                            value={containerConfig.resources.limits.cpu}
                            onChange={(e) =>
                              updateContainer(containerIndex, {
                                resources: {
                                  ...containerConfig.resources,
                                  limits: {
                                    ...containerConfig.resources.limits,
                                    cpu: e.target.value,
                                  },
                                },
                              })
                            }
                          />
                          <Input
                            placeholder="Memory (e.g., 256Mi)"
                            value={containerConfig.resources.limits.memory}
                            onChange={(e) =>
                              updateContainer(containerIndex, {
                                resources: {
                                  ...containerConfig.resources,
                                  limits: {
                                    ...containerConfig.resources.limits,
                                    memory: e.target.value,
                                  },
                                },
                              })
                            }
                          />
                        </div>
                      </div>
                    </div>
                  </div>

                  <div className="space-y-2">
                    <EnvironmentEditor
                      container={containerConfig.container}
                      onUpdate={(updates) =>
                        updateContainer(containerIndex, {
                          container: {
                            ...containerConfig.container,
                            ...updates,
                          },
                        })
                      }
                    />
                  </div>

                  <div className="grid grid-cols-2 gap-4">
                    <div className="space-y-2">
                      <Label htmlFor={`port-${containerIndex}`}>
                        Container Port (optional)
                      </Label>
                      <Input
                        id={`port-${containerIndex}`}
                        type="number"
                        min="1"
                        max="65535"
                        value={containerConfig.port || ''}
                        onChange={(e) =>
                          updateContainer(containerIndex, {
                            port: e.target.value
                              ? parseInt(e.target.value)
                              : undefined,
                          })
                        }
                        placeholder="12306"
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor={`pullPolicy-${containerIndex}`}>
                        Image Pull Policy
                      </Label>
                      <Select
                        value={containerConfig.pullPolicy}
                        onValueChange={(value) =>
                          updateContainer(containerIndex, {
                            pullPolicy: value as
                              | 'Always'
                              | 'IfNotPresent'
                              | 'Never',
                          })
                        }
                      >
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="IfNotPresent">
                            IfNotPresent
                          </SelectItem>
                          <SelectItem value="Always">Always</SelectItem>
                          <SelectItem value="Never">Never</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                  </div>
                </CardContent>
                {containerIndex < formData.containers.length - 1 && (
                  <Separator />
                )}
              </Card>
            ))}
          </div>
        )

      case 3:
        return (
          <div className="space-y-4">
            <h3 className="text-lg font-medium">Review & Edit Configuration</h3>
            <p className="text-sm text-muted-foreground">
              Review and edit the generated YAML configuration before creating
              the deployment. You can modify any part of the configuration
              directly in the editor below.
            </p>
            <SimpleYamlEditor
              value={generateDeploymentYaml()}
              onChange={(value) => setEditedYaml(value || '')}
              disabled={false}
              height="500px"
            />
          </div>
        )

      default:
        return null
    }
  }

  const getStepTitle = () => {
    switch (step) {
      case 1:
        return 'Basic Configuration'
      case 2:
        return 'Containers & Resources'
      case 3:
        return 'Edit YAML & Create'
      default:
        return ''
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleDialogChange}>
      <DialogContent
        className="!max-w-4xl max-h-[90vh] overflow-y-auto sm:!max-w-4xl"
        onPointerDownOutside={(e) => {
          e.preventDefault()
        }}
        onEscapeKeyDown={(e) => {
          e.preventDefault()
        }}
      >
        <DialogHeader>
          <DialogTitle>Create Deployment</DialogTitle>
          <DialogDescription>
            Step {step} of {totalSteps}: {getStepTitle()}
          </DialogDescription>
        </DialogHeader>

        <div className="py-4">
          {/* Progress indicator */}
          <div className="flex justify-between mb-6">
            {Array.from({ length: totalSteps }, (_, i) => i + 1).map(
              (stepNum) => (
                <div
                  key={stepNum}
                  className={`flex items-center ${
                    stepNum < totalSteps ? 'flex-1' : ''
                  }`}
                >
                  <div
                    className={`w-8 h-8 rounded-full flex items-center justify-center text-sm font-medium ${
                      stepNum <= step
                        ? 'bg-primary text-primary-foreground'
                        : 'bg-muted text-muted-foreground'
                    }`}
                  >
                    {stepNum}
                  </div>
                  {stepNum < totalSteps && (
                    <div
                      className={`flex-1 h-0.5 mx-2 ${
                        stepNum < step ? 'bg-primary' : 'bg-muted'
                      }`}
                    />
                  )}
                </div>
              )
            )}
          </div>

          {renderStep()}
        </div>

        <DialogFooter>
          <div className="flex justify-between w-full">
            <div>
              {step > 1 && (
                <Button variant="outline" onClick={handlePrevious}>
                  Previous
                </Button>
              )}
            </div>
            <div className="space-x-2">
              <Button
                variant="outline"
                onClick={() => handleDialogChange(false)}
              >
                Cancel
              </Button>
              {step < totalSteps ? (
                <Button onClick={handleNext} disabled={!validateStep(step)}>
                  Next
                </Button>
              ) : (
                <Button
                  onClick={handleCreate}
                  disabled={!validateStep(step) || isCreating}
                >
                  {isCreating ? 'Creating...' : 'Create Deployment'}
                </Button>
              )}
            </div>
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
