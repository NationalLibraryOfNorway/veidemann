import { DatePipe } from '@angular/common';
import { ChangeDetectionStrategy,ChangeDetectorRef,Component,forwardRef,inject } from '@angular/core';
import {
NG_VALUE_ACCESSOR,
ReactiveFormsModule,
ValidationErrors,
Validators
} from '@angular/forms';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { FlexDirective,LayoutDirective } from '@ngbracket/ngx-layout';
import { LayoutGapDirective } from '@ngbracket/ngx-layout/flex';
import { Observable,of } from 'rxjs';
import { first,map,tap } from 'rxjs/operators';
import { Meta } from '../../../../shared/models';
import { VALID_COLLECTION_NAME } from '../../../../shared/validation/patterns';
import { AnnotationComponent } from '../annotation/annotation.component';
import { LabelComponent } from '../label/label.component';
import { MetaComponent } from '../meta/meta.component';

@Component({
  selector: 'app-collection-meta',
  templateUrl: './collection-meta.component.html',
  styleUrls: ['./collection-meta.component.css'],
  providers: [
    {provide: NG_VALUE_ACCESSOR, useExisting: forwardRef(() => CollectionMetaComponent), multi: true},
    DatePipe
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    MatFormFieldModule,
    MatInputModule,
    ReactiveFormsModule,
    LabelComponent,
    AnnotationComponent,
    FlexDirective,
    LayoutDirective,
    LayoutGapDirective,
  ],
  standalone: true
})
export class CollectionMetaComponent extends MetaComponent {
  private cdr = inject(ChangeDetectorRef);

  protected override createForm() {
    super.createForm();
  }

  protected override updateForm(meta: Meta) {
    this.name.clearValidators();
    this.name.setValidators(
      Validators.compose([
        Validators.required,
        Validators.minLength(2),
        Validators.pattern(VALID_COLLECTION_NAME)
      ])
    );
    super.updateForm(meta);
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
