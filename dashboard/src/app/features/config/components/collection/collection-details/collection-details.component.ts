import { ChangeDetectionStrategy, Component, EventEmitter, Input, OnChanges, Output, SimpleChanges, inject } from '@angular/core';
import {
  Collection,
  ConfigObject,
  Kind,
  Meta,
  RotationPolicy,
  SubCollection,
  SubCollectionType
} from '../../../../../shared/models';
import {
  AbstractControl,
  ReactiveFormsModule,
  UntypedFormBuilder,
  UntypedFormGroup
} from '@angular/forms';
import {AuthService} from '../../../../../core/auth';
import {MatCardModule} from '@angular/material/card';
import {MatIcon} from '@angular/material/icon';
import {CollectionMetaComponent} from '../../collection-meta/collection-meta.component';
import {MatFormFieldModule} from '@angular/material/form-field';
import {MatSelectModule} from '@angular/material/select';
import {FilesizeInputComponent} from '../../filesize-input/filesize-input.component';
import {MatButtonModule} from '@angular/material/button';
import {MatTooltip} from '@angular/material/tooltip';
import {MatInputModule} from '@angular/material/input';
import {CopyIdDirective} from '../../../../../shared/directives';
import {BooleanStateChipComponent} from '../../../../../shared/components';
import {SubcollectionChipsComponent} from '../subcollection-chips/subcollection-chips.component';
import {configKindIcon} from '../../../func/config-kind-icon';


@Component({
  selector: 'app-collection-details',
  templateUrl: './collection-details.component.html',
  styleUrls: ['./collection-details.component.css', '../../config-details-grid.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    CopyIdDirective,
    BooleanStateChipComponent,
    SubcollectionChipsComponent,
    MatCardModule,
    ReactiveFormsModule,
    MatIcon,
    CollectionMetaComponent,
    MatFormFieldModule,
    MatInputModule,
    MatSelectModule,
    FilesizeInputComponent,
    MatTooltip,
    MatButtonModule,
  ],
  standalone: true
})
export class CollectionDetailsComponent implements OnChanges {
  readonly configKindIcon = configKindIcon;
  protected fb = inject(UntypedFormBuilder);
  protected authService = inject(AuthService);


  constructor() {
    this.createForm();
  }

  get canEdit(): boolean {
    return this.authService.canUpdate(this.configObject.kind);
  }

  get showSave(): boolean {
    return (this.configObject && !this.configObject.id);
  }

  get canSave(): boolean {
    return this.form.valid;
  }

  get canUpdate(): boolean {
    return (this.form.valid && this.form.dirty);
  }

  get canRevert(): boolean {
    return this.form.dirty;
  }

  get fileSize(): AbstractControl {
    return this.form.get('fileSize');
  }

  get validFileSize(): boolean {
    if (this.fileSize.value === 0) {
      return false;
    }
    return true;
  }

  readonly RotationPolicy = RotationPolicy;
  readonly SubCollectionType = SubCollectionType;

  @Input()
  configObject: ConfigObject;

  @Input()
  rotationPolicies: RotationPolicy[] = [];

  @Input()
  subCollectionTypes: SubCollectionType[] = [];

  @Output()
  save = new EventEmitter<ConfigObject>();

  @Output()
  update = new EventEmitter<ConfigObject>();

  // noinspection ReservedWordAsName
  form: UntypedFormGroup;

  ngOnChanges(changes: SimpleChanges) {
    if (changes['configObject']) {
      if (this.configObject) {
        this.updateForm();
      } else {
        this.form.reset();
      }
    }
  }

  onSave() {
    this.save.emit(this.prepareSave());
  }

  onUpdate() {
    this.update.emit(this.prepareSave());
  }

  onRevert(): void {
    this.updateForm();
  }

  protected createForm(): void {
    this.form = this.fb.group({
      id: '',
      collectionDedupPolicy: '',
      fileRotationPolicy: '',
      compress: '',
      fileSize: '',
      subCollectionsList: [[]],
      meta: new Meta()
    });
  }

  protected updateForm(): void {
    this.form.patchValue({
      id: this.configObject.id,
      meta: this.configObject.meta,
      collectionDedupPolicy: this.configObject.collection.collectionDedupPolicy || RotationPolicy.NONE,
      fileRotationPolicy: this.configObject.collection.fileRotationPolicy || RotationPolicy.NONE,
      compress: this.configObject.collection.compress,
      fileSize: this.configObject.collection.fileSize || 1073741824,
      subCollectionsList: this.configObject.collection.subCollectionsList,
    });
    this.form.markAsPristine();
    this.form.markAsUntouched();
    if (!this.canEdit) {
      this.form.disable();
    }
  }

  protected prepareSave(): ConfigObject {
    const formModel = this.form.value;

    const configObject = new ConfigObject({kind: Kind.COLLECTION});
    if (this.configObject.id !== '') {
      configObject.id = this.configObject.id;
    }

    const collection = new Collection();
    collection.collectionDedupPolicy = formModel.collectionDedupPolicy;
    collection.fileRotationPolicy = formModel.fileRotationPolicy;
    collection.compress = formModel.compress;
    collection.fileSize = formModel.fileSize || 0;
    collection.subCollectionsList = formModel.subCollectionsList.map(sub => new SubCollection(sub));

    configObject.meta = formModel.meta;
    configObject.collection = collection;

    return configObject;
  }

}
