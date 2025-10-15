import {ChangeDetectionStrategy, Component, Inject, OnInit} from '@angular/core';
import {CollectionDetailsComponent} from '..';
import {ReactiveFormsModule, UntypedFormBuilder} from '@angular/forms';
import {AuthService} from '../../../../../core/auth';
import {MAT_DIALOG_DATA, MatDialogModule, MatDialogRef} from '@angular/material/dialog';
import {ConfigDialogData} from '../../../func';
import {ConfigObject} from '../../../../../shared/models/config';
import {MatFormFieldModule} from '@angular/material/form-field';
import {MatSelectModule} from '@angular/material/select';
import {CollectionMetaComponent} from '../../collection-meta/collection-meta.component';
import {MatCheckbox} from '@angular/material/checkbox';
import {FilesizeInputComponent} from '../../filesize-input/filesize-input.component';
import {MatIcon} from '@angular/material/icon';
import {MatButtonModule} from '@angular/material/button';
import {MatTooltip} from '@angular/material/tooltip';

@Component({
  selector: 'app-collection-dialog',
  templateUrl: './collection-dialog.component.html',
  styleUrls: ['./collection-dialog.component.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    MatDialogModule,
    MatFormFieldModule,
    MatSelectModule,
    CollectionMetaComponent,
    ReactiveFormsModule,
    MatCheckbox,
    FilesizeInputComponent,
    MatIcon,
    MatButtonModule,
    MatTooltip

  ],
  standalone: true
})
export class CollectionDialogComponent extends CollectionDetailsComponent implements OnInit {

  constructor(protected override fb: UntypedFormBuilder,
              protected override authService: AuthService,
              @Inject(MAT_DIALOG_DATA) public data: ConfigDialogData,
              public dialogRef: MatDialogRef<CollectionDialogComponent>) {
    super(fb, authService);
    this.createForm();
    this.configObject = this.data.configObject;
    this.rotationPolicies = this.data.options.rotationPolicies;
    this.subCollectionTypes = this.data.options.subCollectionTypes;
  }

  ngOnInit(): void {
    this.updateForm();
  }

  onDialogClose(): ConfigObject {
    return this.prepareSave();
  }

}
