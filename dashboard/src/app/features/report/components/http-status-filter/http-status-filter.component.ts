import {ChangeDetectionStrategy, Component, EventEmitter, Input, Output} from '@angular/core';
import {MatChipsModule} from '@angular/material/chips';
import {MatTooltipModule} from '@angular/material/tooltip';

export type HttpStatusFamily = 1 | 2 | 3 | 4 | 5;

interface HttpStatusOption {
  family: HttpStatusFamily;
  label: string;
  description: string;
}

@Component({
  selector: 'app-http-status-filter',
  template: `
    <fieldset>
      <legend i18n="@@httpStatusFilterLegend">Status code</legend>
      <mat-chip-listbox multiple
        aria-label="Filter by status code"
        i18n-aria-label="@@httpStatusFilterLabel"
        [value]="selectedValues"
        (change)="onChange($event.value)">
        @for (option of visibleOptions; track option.family) {
          <mat-chip-option [value]="option.family"
            [aria-label]="option.description"
            [matTooltip]="option.description">
            {{option.label}}
          </mat-chip-option>
        }
        @for (statusCode of visibleStatusCodes; track statusCode) {
          <mat-chip-option [value]="statusCode"
            [aria-label]="exactStatusDescription(statusCode)"
            [matTooltip]="exactStatusDescription(statusCode)">
            {{statusCode}}
          </mat-chip-option>
        }
      </mat-chip-listbox>
    </fieldset>
  `,
  styles: [`
    :host { display: block; min-width: 0; }
    fieldset { min-width: 0; margin: 0; padding: 0; border: 0; }
    legend {
      margin-bottom: 8px;
      color: var(--mat-sys-on-surface-variant);
      font: var(--mat-sys-label-medium);
    }
  `],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [MatChipsModule, MatTooltipModule],
  standalone: true,
})
export class HttpStatusFilterComponent {
  @Input() value: readonly HttpStatusFamily[] = [];
  @Input() families: readonly HttpStatusFamily[] = [1, 2, 3, 4, 5];
  @Input() exactValue: readonly number[] = [];
  @Input() statusCodes: readonly number[] = [];
  @Output() readonly valueChange = new EventEmitter<HttpStatusFamily[]>();
  @Output() readonly exactValueChange = new EventEmitter<number[]>();

  readonly options: readonly HttpStatusOption[] = [
    {
      family: 1,
      label: '1xx',
      description: $localize`:@@httpStatusInformationalDescription:Informational responses (100 – 199)`,
    },
    {
      family: 2,
      label: '2xx',
      description: $localize`:@@httpStatusSuccessfulDescription:Successful responses (200 – 299)`,
    },
    {
      family: 3,
      label: '3xx',
      description: $localize`:@@httpStatusRedirectionDescription:Redirection messages (300 – 399)`,
    },
    {
      family: 4,
      label: '4xx',
      description: $localize`:@@httpStatusClientErrorDescription:Client error responses (400 – 499)`,
    },
    {
      family: 5,
      label: '5xx',
      description: $localize`:@@httpStatusServerErrorDescription:Server error responses (500 – 599)`,
    },
  ];

  get visibleOptions(): readonly HttpStatusOption[] {
    const families = new Set(this.families);
    return this.options.filter(option => families.has(option.family));
  }

  get visibleStatusCodes(): readonly number[] {
    return [...new Set(this.statusCodes)]
      .filter(isHttpStatusCode)
      .sort((left, right) => left - right);
  }

  get selectedValues(): readonly number[] {
    return [...this.value, ...this.exactValue];
  }

  exactStatusDescription(statusCode: number): string {
    return $localize`:@@httpStatusExactDescription:Status code ${statusCode}:STATUS_CODE:`;
  }

  onChange(value: number[] | null): void {
    const selection = value ?? [];
    this.valueChange.emit(selection.filter(isHttpStatusFamily));
    this.exactValueChange.emit(selection.filter(isHttpStatusCode));
  }
}

export function httpStatusFamily(statusCode: number): HttpStatusFamily | null {
  const family = Math.trunc(statusCode / 100);
  return family >= 1 && family <= 5 ? family as HttpStatusFamily : null;
}

function isHttpStatusFamily(value: number): value is HttpStatusFamily {
  return Number.isInteger(value) && value >= 1 && value <= 5;
}

function isHttpStatusCode(value: number): boolean {
  return Number.isInteger(value) && value >= 100 && value <= 599;
}

export function uniqueHttpStatusCodes(values: readonly number[]): number[] {
  return [...new Set(values.filter(isHttpStatusCode))].sort((left, right) => left - right);
}
