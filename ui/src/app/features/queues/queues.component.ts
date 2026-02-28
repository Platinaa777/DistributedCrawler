import { Component, OnDestroy, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { TableModule } from 'primeng/table';
import { ButtonModule } from 'primeng/button';
import { DialogModule } from 'primeng/dialog';
import { InputTextModule } from 'primeng/inputtext';
import { SelectModule } from 'primeng/select';
import { CheckboxModule } from 'primeng/checkbox';
import { CardModule } from 'primeng/card';
import { TabsModule } from 'primeng/tabs';
import { ProgressSpinnerModule } from 'primeng/progressspinner';
import { Router } from '@angular/router';
import { Subject } from 'rxjs';
import { takeUntil } from 'rxjs/operators';
import { QueueAdminApiService } from '../../core/services/api/queue-admin-api.service';
import { QueueEndpoint, QueueRoutingRule, QueueStage, QueueBrokerType } from '../../core/models/queue.model';

const emptyEndpoint = (): Partial<QueueEndpoint> => ({
  display_name: '',
  broker_type: 'QUEUE_BROKER_TYPE_RABBITMQ',
  stage: 'QUEUE_STAGE_CRAWL',
  host: '',
  queue_name: '',
  secret_key: ''
});

@Component({
  selector: 'app-queues',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    TableModule,
    ButtonModule,
    DialogModule,
    InputTextModule,
    SelectModule,
    CheckboxModule,
    CardModule,
    TabsModule,
    ProgressSpinnerModule
  ],
  template: `
    <div class="container mx-auto p-6">
      <div class="flex justify-between items-center mb-6">
        <div>
          <h1 class="text-3xl font-bold text-gray-900 dark:text-white">Queue Admin</h1>
          <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">Manage queue endpoints and routing rules.</p>
        </div>
        <p-button [outlined]="true" severity="secondary" (onClick)="goBack()">
          <i class="pi pi-arrow-left mr-2"></i>Back to Jobs
        </p-button>
      </div>

      <p-card *ngIf="error" styleClass="bg-red-50 p-4 mb-4">
        <p class="text-red-700">{{ error }}</p>
      </p-card>

      <p-tabs [(value)]="activeTab">
        <p-tabpanel header="Queue Endpoints" value="0">
          <div class="flex justify-end mb-4">
            <p-button label="Add Endpoint" icon="pi pi-plus" (onClick)="openCreateDialog()" />
          </div>

          <p-card *ngIf="loadingEndpoints" styleClass="text-center p-8">
            <p-progressSpinner />
          </p-card>

          <p-table *ngIf="!loadingEndpoints" [value]="endpoints" [tableStyle]="{'min-width': '60rem'}">
            <ng-template pTemplate="header">
              <tr>
                <th>Name</th>
                <th>Stage</th>
                <th>Broker</th>
                <th>Host</th>
                <th>Queue/Topic</th>
                <th>Actions</th>
              </tr>
            </ng-template>
            <ng-template pTemplate="body" let-ep>
              <tr>
                <td>{{ ep.display_name }}</td>
                <td>{{ stageLabel(ep.stage) }}</td>
                <td>{{ brokerLabel(ep.broker_type) }}</td>
                <td class="font-mono text-sm">{{ ep.host }}</td>
                <td class="font-mono text-sm">{{ ep.queue_name }}</td>
                <td>
                  <div class="flex gap-2">
                    <p-button icon="pi pi-pencil" size="small" [outlined]="true" (onClick)="openEditDialog(ep)" />
                    <p-button icon="pi pi-trash" size="small" severity="danger" [outlined]="true" (onClick)="deleteEndpoint(ep)" />
                  </div>
                </td>
              </tr>
            </ng-template>
            <ng-template pTemplate="emptymessage">
              <tr>
                <td colspan="6" class="text-center p-8 text-gray-500 dark:text-gray-400">
                  <i class="pi pi-server text-6xl block mb-4"></i>
                  <p>No queue endpoints configured.</p>
                </td>
              </tr>
            </ng-template>
          </p-table>
        </p-tabpanel>

        <p-tabpanel header="Routing Rules" value="1">
          <div class="mb-4 flex justify-between items-center">
            <div class="flex gap-4 items-center">
              <label class="font-medium">Stage:</label>
              <p-select
                [options]="stageOptions"
                [(ngModel)]="selectedStage"
                (onChange)="loadRoutingRules()"
                optionLabel="label"
                optionValue="value"
                styleClass="w-48" />
            </div>
            <p-button label="Create Rule" icon="pi pi-plus" (onClick)="openRuleDialog()" />
          </div>

          <p-card *ngIf="loadingRules" styleClass="text-center p-8">
            <p-progressSpinner />
          </p-card>

          <div *ngIf="!loadingRules">
            <div *ngIf="routingRules.length === 0" class="text-center p-8 text-gray-500">
              <i class="pi pi-shuffle text-6xl block mb-4"></i>
              <p>No routing rules for this stage.</p>
            </div>

            <div *ngFor="let rule of routingRules">
              <p-card styleClass="mb-4">
                <div class="flex justify-between items-center">
                  <div>
                    <span class="font-semibold">{{ stageLabel(rule.stage) }}</span>
                    <span class="text-gray-500 ml-2">(scope: {{ rule.scope }})</span>
                  </div>
                  <p-button label="Edit" icon="pi pi-pencil" [outlined]="true" size="small" (onClick)="openRuleEditDialog(rule)" />
                </div>
              </p-card>
            </div>
          </div>
        </p-tabpanel>
      </p-tabs>
    </div>

    <!-- Endpoint Dialog -->
    <p-dialog
      [(visible)]="showEndpointDialog"
      [header]="editingEndpoint?.id ? 'Edit Endpoint' : 'New Endpoint'"
      [modal]="true"
      [style]="{width: '720px'}"
      [closable]="true">
      <div *ngIf="editingEndpoint" class="flex flex-col gap-4 pt-2">
        <div class="flex flex-col gap-1">
          <label class="font-medium text-sm">Display Name</label>
          <input pInputText [(ngModel)]="editingEndpoint!.display_name" placeholder="e.g. RabbitMQ US-East" />
        </div>
        <div class="flex gap-4">
          <div class="flex flex-col gap-1 flex-1">
            <label class="font-medium text-sm">Stage</label>
            <p-select [options]="stageOptions" [(ngModel)]="editingEndpoint!.stage" optionLabel="label" optionValue="value" />
          </div>
          <div class="flex flex-col gap-1 flex-1">
            <label class="font-medium text-sm">Broker</label>
            <p-select [options]="brokerOptions" [(ngModel)]="editingEndpoint!.broker_type" optionLabel="label" optionValue="value" />
          </div>
        </div>
        <div class="flex flex-col gap-1">
          <label class="font-medium text-sm">Host</label>
          <input pInputText [(ngModel)]="editingEndpoint!.host" placeholder="e.g. rabbitmq.svc:5672" />
        </div>
        <div class="flex flex-col gap-1">
          <label class="font-medium text-sm">Queue / Topic Name</label>
          <input pInputText [(ngModel)]="editingEndpoint!.queue_name" placeholder="e.g. crawl_queue_us" />
        </div>
        <div class="flex flex-col gap-1">
          <label class="font-medium text-sm">Secret Key</label>
          <input pInputText [(ngModel)]="editingEndpoint!.secret_key" placeholder="e.g. rmq_default" />
        </div>
      </div>
      <ng-template pTemplate="footer">
        <p-button label="Cancel" severity="secondary" [outlined]="true" (onClick)="showEndpointDialog = false" />
        <p-button label="Save" [loading]="saving" (onClick)="saveEndpoint()" />
      </ng-template>
    </p-dialog>

    <!-- Routing Rule Dialog -->
    <p-dialog
      [(visible)]="showRuleDialog"
      header="Routing Rule"
      [modal]="true"
      [style]="{width: '520px'}"
      [closable]="true">
      <div *ngIf="editingRule" class="flex flex-col gap-4 pt-2">
        <div class="flex gap-4">
          <div class="flex flex-col gap-1 flex-1">
            <label class="font-medium text-sm">Stage</label>
            <p-select [options]="stageOptions" [(ngModel)]="editingRule!.stage" optionLabel="label" optionValue="value" />
          </div>
          <div class="flex flex-col gap-1 flex-1">
            <label class="font-medium text-sm">Scope</label>
            <input pInputText [(ngModel)]="editingRule!.scope" placeholder="global" />
          </div>
        </div>
      </div>
      <ng-template pTemplate="footer">
        <p-button label="Cancel" severity="secondary" [outlined]="true" (onClick)="showRuleDialog = false" />
        <p-button label="Save" [loading]="saving" (onClick)="saveRule()" />
      </ng-template>
    </p-dialog>
  `,
  styles: [':host { display: block; }']
})
export class QueuesComponent implements OnInit, OnDestroy {
  private destroy$ = new Subject<void>();

  endpoints: QueueEndpoint[] = [];
  routingRules: QueueRoutingRule[] = [];
  loadingEndpoints = false;
  loadingRules = false;
  saving = false;
  error: string | null = null;
  activeTab = '0';

  showEndpointDialog = false;
  editingEndpoint: Partial<QueueEndpoint> | null = null;

  showRuleDialog = false;
  editingRule: Partial<QueueRoutingRule> | null = null;

  selectedStage: QueueStage = 'QUEUE_STAGE_CRAWL';

  readonly stageOptions = [
    { label: 'Crawl', value: 'QUEUE_STAGE_CRAWL' as QueueStage },
    { label: 'Parse', value: 'QUEUE_STAGE_PARSE' as QueueStage }
  ];

  readonly brokerOptions = [
    { label: 'RabbitMQ', value: 'QUEUE_BROKER_TYPE_RABBITMQ' as QueueBrokerType },
    { label: 'Kafka', value: 'QUEUE_BROKER_TYPE_KAFKA' as QueueBrokerType }
  ];

  constructor(
    private api: QueueAdminApiService,
    private router: Router
  ) {}

  ngOnInit(): void {
    this.loadEndpoints();
    this.loadRoutingRules();
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
  }

  loadEndpoints(): void {
    this.loadingEndpoints = true;
    this.api.listEndpoints().pipe(takeUntil(this.destroy$)).subscribe({
      next: res => {
        this.endpoints = res.endpoints || [];
        this.loadingEndpoints = false;
      },
      error: err => {
        this.error = `Failed to load endpoints: ${err.message}`;
        this.loadingEndpoints = false;
      }
    });
  }

  loadRoutingRules(): void {
    this.loadingRules = true;
    this.api.listRoutingRules(this.selectedStage).pipe(takeUntil(this.destroy$)).subscribe({
      next: res => {
        this.routingRules = res.rules || [];
        this.loadingRules = false;
      },
      error: err => {
        this.error = `Failed to load routing rules: ${err.message}`;
        this.loadingRules = false;
      }
    });
  }

  openCreateDialog(): void {
    this.editingEndpoint = emptyEndpoint();
    this.showEndpointDialog = true;
  }

  openEditDialog(ep: QueueEndpoint): void {
    this.editingEndpoint = { ...ep };
    this.showEndpointDialog = true;
  }

  saveEndpoint(): void {
    if (!this.editingEndpoint) return;
    this.saving = true;
    const obs = this.editingEndpoint.id
      ? this.api.updateEndpoint(this.editingEndpoint as QueueEndpoint)
      : this.api.createEndpoint(this.editingEndpoint);

    obs.pipe(takeUntil(this.destroy$)).subscribe({
      next: () => {
        this.saving = false;
        this.showEndpointDialog = false;
        this.loadEndpoints();
      },
      error: err => {
        this.error = `Failed to save endpoint: ${err.message}`;
        this.saving = false;
      }
    });
  }

  deleteEndpoint(ep: QueueEndpoint): void {
    if (!confirm(`Delete endpoint "${ep.display_name}"?`)) return;
    this.api.deleteEndpoint(ep.id!).pipe(takeUntil(this.destroy$)).subscribe({
      next: () => this.loadEndpoints(),
      error: err => { this.error = `Failed to delete: ${err.message}`; }
    });
  }

  openRuleDialog(): void {
    this.editingRule = {
      stage: this.selectedStage,
      scope: 'global'
    };
    this.showRuleDialog = true;
  }

  openRuleEditDialog(rule: QueueRoutingRule): void {
    this.editingRule = { ...rule };
    this.showRuleDialog = true;
  }

  saveRule(): void {
    if (!this.editingRule) return;
    this.saving = true;
    this.api.upsertRoutingRule(this.editingRule).pipe(takeUntil(this.destroy$)).subscribe({
      next: () => {
        this.saving = false;
        this.showRuleDialog = false;
        this.loadRoutingRules();
      },
      error: err => {
        this.error = `Failed to save rule: ${err.message}`;
        this.saving = false;
      }
    });
  }

  stageLabel(stage: QueueStage): string {
    return stage === 'QUEUE_STAGE_CRAWL' ? 'Crawl' : stage === 'QUEUE_STAGE_PARSE' ? 'Parse' : stage;
  }

  brokerLabel(bt: QueueBrokerType): string {
    return bt === 'QUEUE_BROKER_TYPE_RABBITMQ' ? 'RabbitMQ' : bt === 'QUEUE_BROKER_TYPE_KAFKA' ? 'Kafka' : bt;
  }

  goBack(): void {
    this.router.navigate(['/jobs']);
  }
}
