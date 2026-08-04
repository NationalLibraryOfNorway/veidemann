import { ChangeDetectionStrategy,ChangeDetectorRef,Component,EventEmitter,forwardRef,inject,Input,Output } from '@angular/core';

import { CdkTextareaAutosize } from '@angular/cdk/text-field';
import { DatePipe } from '@angular/common';
import {
AbstractControl,
AsyncValidator,
NG_ASYNC_VALIDATORS,
NG_VALUE_ACCESSOR,
ReactiveFormsModule,
ValidationErrors,
Validators
} from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIcon } from '@angular/material/icon';
import { MatInput } from '@angular/material/input';
import { MatListModule } from '@angular/material/list';
import { MatProgressBar } from '@angular/material/progress-bar';
import { MatTooltip } from '@angular/material/tooltip';
import { RouterLink } from '@angular/router';
import { Observable,of } from 'rxjs';
import { first,map,tap } from 'rxjs/operators';
import { ConfigApiService } from '../../../../core';
import { ConfigObject,ConfigRef,Meta } from '../../../../shared/models';
import { SeedUrlValidator } from '../../../../shared/validation/existing-url-validation';
import { AnnotationComponent } from '../annotation/annotation.component';
import { LabelComponent } from '../label/label.component';
import { MetaComponent } from '../meta/meta.component';
import { validUrlValidator } from './seed-urlvalidation';

export interface Parcel {
  seed: ConfigObject | ConfigObject[];
  entityRef: ConfigRef;
}

@Component({
  selector: 'app-seed-meta',
  templateUrl: './seed-meta.component.html',
  styleUrls: ['./seed-meta.component.css'],
  providers: [
    DatePipe,
    {provide: NG_VALUE_ACCESSOR, useExisting: forwardRef(() => SeedMetaComponent), multi: true},
    {provide: NG_ASYNC_VALIDATORS, useExisting: forwardRef(() => SeedMetaComponent), multi: true}
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    CdkTextareaAutosize,
    MatButtonModule,
    MatFormFieldModule,
    MatIcon,
    MatInput,
    MatListModule,
    MatProgressBar,
    MatTooltip,
    ReactiveFormsModule,
    RouterLink,
    LabelComponent,
    AnnotationComponent,

  ],
  standalone: true
})
export class SeedMetaComponent extends MetaComponent implements AsyncValidator {
  private cdr = inject(ChangeDetectorRef);
  private configApiService = inject(ConfigApiService);


  @Input()
  entityRef: ConfigRef;

  @Output()
  move = new EventEmitter<Parcel>();

  private asyncUrlValidator: (entityRef: ConfigRef) => (control: AbstractControl) => Observable<ValidationErrors | null>;

  constructor() {

    super();

    this.asyncUrlValidator = SeedUrlValidator.createBackendValidator(this.configApiService);
  }

  get isSingleUrl(): boolean {
    const url = this.name.value;
    const parts = url.split(/[\s]+/);
    if (parts.length > 1) {
      for (let i = 1; i < parts.length; i++) {
        if (parts[i] !== '') {
          return false;
        }
      }
    }
    return true;
  }

  protected override createForm(): void {
    super.createForm();
  }

  override updateForm(meta: Meta): void {
    this.name.clearValidators();
    this.name.clearAsyncValidators();
    this.name.setValidators(Validators.compose([Validators.required, validUrlValidator]));
    if (!meta.created) {
      this.name.setAsyncValidators(this.asyncUrlValidator(this.entityRef));
    }
    super.updateForm(meta);
  }

  onRemoveExistingUrl(seed: ConfigObject) {
    let value = '';
    const match = this.name.value.includes(seed.meta.name) > 0;
    if (match) {
      value = this.name.value.replace(seed.meta.name, '').trim();
      this.name.setValue(value);
    } else {
      const url = new URL(seed.meta.name);
      const domain = url.hostname.replace('www.', '');
      const urls = this.name.value.trim().split(/\s+/);
      const expression = new RegExp(`.*(${domain}).*`);
      const found = urls.findIndex(u => expression.test(u));
      if (found > -1) {
        urls.splice(found, 1);
      }
      value = urls.join('\n');
      this.name.setValue(value);
    }
    if (!value) {
      this.form.markAsPristine();
      this.form.markAsUntouched();
    }
  }

  onRemoveExistingUrls(seeds: ConfigObject[]) {
    const urls: string[] = this.name.value.trim().split(/\s+/);
    for (const seed of seeds) {
      const url = new URL(seed.meta.name);
      const domain = url.hostname.replace('www.', '');
      const expression = new RegExp(`.*(${domain}).*`);
      const found = urls.findIndex(u => expression.test(u));
      if (found > -1) {
        urls.splice(found, 1);
      }
    }
    const value: string = urls.join('\n');
    this.name.setValue(value);

    if (!value) {
      this.form.markAsPristine();
      this.form.markAsUntouched();
    }
  }

  onMoveSeedToCurrentEntity(seed: ConfigObject) {
    this.onRemoveExistingUrl(seed);
    this.move.emit({seed, entityRef: this.entityRef});
  }

  onMoveSeedsToCurrentEntity(seeds: ConfigObject[]) {
    this.onRemoveExistingUrls(seeds);
    this.move.emit({seed: seeds, entityRef: this.entityRef});
  }

  goToUrl(url: string): void {
    window.open(url, '_blank');
  }

  override validate(): Promise<ValidationErrors | null> | Observable<ValidationErrors | null> {
    return (this.name.pending
        ? this.name.statusChanges.pipe(
          map(state => state === 'VALID' ? null : this.name.errors),
          tap(() => this.cdr.markForCheck()))
        : this.name.valid
          ? of(null)
          : of(this.name.errors)
    ).pipe(
      first() // must ensure the observable returned is completed
    );
  }
}
