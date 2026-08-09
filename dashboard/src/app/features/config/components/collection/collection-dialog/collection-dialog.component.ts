import { ChangeDetectionStrategy,Component,OnInit,inject } from '@angular/core';
import { ReactiveFormsModule } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MAT_DIALOG_DATA,MatDialogModule,MatDialogRef } from '@angular/material/dialog';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { MatTooltip } from '@angular/material/tooltip';
import { CollectionDetailsComponent } from '..';
import { ConfigObject } from '../../../../../shared/models/config';
import { ConfigDialogData } from '../../../func';
import { CollectionMetaComponent } from '../../collection-meta/collection-meta.component';
import { FilesizeInputComponent } from '../../filesize-input/filesize-input.component';
import {BooleanStateChipComponent} from '../../../../../shared/components';
import {SubcollectionChipsComponent} from '../subcollection-chips/subcollection-chips.component';

@Component({
  selector: 'app-collection-dialog',
  templateUrl: './collection-dialog.component.html',
  styleUrls: ['./collection-dialog.component.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    FilesizeInputComponent,
    BooleanStateChipComponent,
    SubcollectionChipsComponent,
    MatButtonModule,
    MatCardModule,
    MatDialogModule,
    MatFormFieldModule,
    MatInputModule,
    MatSelectModule,
    MatTooltip,
    CollectionMetaComponent,
    ReactiveFormsModule,
  ],
  standalone: true
})
export class CollectionDialogComponent extends CollectionDetailsComponent implements OnInit {
  data = inject<ConfigDialogData>(MAT_DIALOG_DATA);
  dialogRef = inject<MatDialogRef<CollectionDialogComponent>>(MatDialogRef);


  constructor() {

    super();

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
