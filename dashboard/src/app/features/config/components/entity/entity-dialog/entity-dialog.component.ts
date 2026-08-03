import { ChangeDetectionStrategy,Component,OnInit,inject } from '@angular/core';
import { ReactiveFormsModule } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MAT_DIALOG_DATA,MatDialogModule,MatDialogRef } from '@angular/material/dialog';
import { EntityDetailsComponent } from '..';
import { ConfigObject } from '../../../../../shared/models/config';
import { ConfigDialogData } from '../../../func';
import { MetaComponent } from '../../meta/meta.component';

@Component({
  selector: 'app-entity-dialog',
  templateUrl: './entity-dialog.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    MatButtonModule,
    MatDialogModule,
    MetaComponent,
    ReactiveFormsModule,
  ],
  standalone: true
})
export class EntityDialogComponent extends EntityDetailsComponent implements OnInit {
  data = inject<ConfigDialogData>(MAT_DIALOG_DATA);
  dialogRef = inject<MatDialogRef<EntityDialogComponent>>(MatDialogRef);

  constructor() {

    super();

    this.createForm();
    this.configObject = this.data.configObject;
  }

  ngOnInit(): void {
    this.updateForm();
  }

  onDialogClose(): ConfigObject {
    return this.prepareSave();
  }
}
